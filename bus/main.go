package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
)

var clients = []*listenerClient{}
var msgsInMem = []EventBusMessage{}
var writeEvery = 2 * time.Second
var deadTime = 10 * time.Second
var isWriting = false

type listenerClient struct {
	conn net.Conn
	id   uuid.UUID
}

func newListenerClient(conn net.Conn) *listenerClient {
	return &listenerClient{
		conn: conn,
		id:   uuid.New(),
	}
}

type EventBusMessage struct {
	Topic    string    `json:"topic"`
	SenderId string    `json:"client_id"`
	Body     string    `json:"body"`
	SentAt   time.Time `json:"sent_at"`
}

func main() {
	fmt.Println("starting tcp client")
	listener, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Err starting a tcp listener:", err)
		return
	}

	fmt.Println("tcp listening")
	wallChan := make(chan []byte)

	go func() {
		for {
			fmt.Println("curr clients:", len(clients))
			time.Sleep(3 * time.Second)
			pruneConnections()
			fmt.Println("pruned clients, new len", len(clients))
		}
	}()

	go writeToInMemWall(wallChan)

	go func() {
		for {
			time.Sleep(writeEvery)
			fmt.Println("writing to wall")
			err := writeToWall()
			if err != nil {
				fmt.Println("err writing to wall: ", err)
			}
			fmt.Println("done writing to wall")
		}
	}()

	defer listener.Close()
	for {
		con, err := listener.Accept()
		if err != nil {
			fmt.Println("Err accepting a connection: ", err)
			continue
		}
		clientConn := newListenerClient(con)
		clients = append(clients, clientConn)

		go handleClient(clientConn, wallChan)
	}
}

func writeToInMemWall(wallChan chan []byte) error {
	for {
		select {
		case msgBts, ok := <-wallChan:
			if !ok {
				continue
			}

			message := EventBusMessage{}
			err := json.Unmarshal(msgBts, &message)
			if err != nil {
				return err
			}

			msgsInMem = append(msgsInMem, message)

		}
	}
}

func handleClient(listener *listenerClient, wallChan chan []byte) {
	defer listener.conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, err := listener.conn.Read(buffer)
		if err != nil {
			fmt.Println("err reading from tcp conn:", err)
			return
		}

		log.Default().Printf("Got from conn: %s\n", buffer[:n])
		for _, client := range clients {
			if client.id != listener.id {
				_, err := client.conn.Write(buffer[:n])
				if err != nil {
					fmt.Println("err broadcasting", err)
				}
				wallChan <- buffer[:n]
			}
		}
	}
}

func pruneConnections() {
	aliveClients := []*listenerClient{}
	for _, client := range clients {
		_, err := client.conn.Write([]byte("ping"))
		if err != nil {
			continue
		}

		aliveClients = append(aliveClients, client)
	}

	clients = aliveClients
}

type wallMessages struct {
	Messages []EventBusMessage `json:"messages"`
}

func writeToWall() error {
	data, err := os.ReadFile("wall.json")
	if err != nil {
		fmt.Println("err reading from wall")
		return err
	}

	wall := wallMessages{}
	err = json.Unmarshal(data, &wall)
	if err != nil {
		fmt.Println("err marshaling wall")
		return err
	}

	removedDeadOnes := []EventBusMessage{}
	for _, msg := range wall.Messages {
		if msg.SentAt.Add(deadTime).Unix() < time.Now().Unix() {
			continue
		}

		removedDeadOnes = append(removedDeadOnes, msg)
	}

	for _, msg := range msgsInMem {
		removedDeadOnes = append(removedDeadOnes, msg)
	}

	inBts, err := json.Marshal(wallMessages{Messages: removedDeadOnes})
	if err != nil {
		return err
	}

	err = os.WriteFile("wall.json", inBts, 0777)
	if err != nil {
		fmt.Println("failed to write to wall part")
		return err
	}

	msgsInMem = []EventBusMessage{}

	return nil
}
