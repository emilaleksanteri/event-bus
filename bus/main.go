package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/google/uuid"
)

var clients = []*listenerClient{}
var msgsInMem = []EventBusMessage{}
var wallSyncIntreval = 10 * time.Second
var deadTime = 20 * time.Second
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
	Id       uuid.UUID
}

func main() {
	fmt.Println("starting tcp client")
	listener, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Err starting a tcp listener:", err)
		return
	}
	defer listener.Close()

	fmt.Println("tcp listening")
	err = recoverInMemWall()
	if err != nil {
		fmt.Println("could not recover in mem wall:", err)
		os.Exit(1)
	}
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
			time.Sleep(wallSyncIntreval)
			err := writeToWall()
			if err != nil {
				fmt.Println("err writing to wall: ", err)
			}
		}
	}()

	go func() {
		quitChan := make(chan os.Signal, 1)
		signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM)
		s := <-quitChan
		log.Default().Println("caught quit signal", s.String())
		writeToWall()
		listener.Close()
		os.Exit(0)
	}()

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

	return
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
			message.Id = uuid.New()

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

func listenerCatchUp(listener *listenerClient) {}

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
	log.Default().Println("Writing to wall.")
	data, err := os.ReadFile("wall.json")
	if err != nil {
		return err
	}

	wall := wallMessages{}
	err = json.Unmarshal(data, &wall)
	if err != nil {
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
		if msg.SentAt.Add(deadTime).Unix() < time.Now().Unix() {
			continue
		}

		contains := slices.ContainsFunc(removedDeadOnes, func(m EventBusMessage) bool {
			if m.Id == msg.Id {
				return true
			}

			return false
		})

		if !contains {
			removedDeadOnes = append(removedDeadOnes, msg)
		}
	}

	inBts, err := json.Marshal(wallMessages{Messages: removedDeadOnes})
	if err != nil {
		return err
	}

	err = os.WriteFile("wall.json", inBts, 0777)
	if err != nil {
		return err
	}

	msgsInMem = removedDeadOnes

	log.Default().Println("Written to wall.")

	return nil
}

func recoverInMemWall() error {
	data, err := os.ReadFile("wall.json")
	if err != nil {
		return err
	}

	wall := wallMessages{}
	err = json.Unmarshal(data, &wall)
	if err != nil {
		return err
	}

	msgsInMem = wall.Messages
	return nil
}
