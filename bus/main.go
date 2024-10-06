package main

import (
	"encoding/json"
	"event-bus/listener"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

var clients = []*listener.ListenerClient{}
var msgsInMem = []listener.EventBusMessage{}
var wallSyncIntreval = 10 * time.Second
var deadTime = 60 * time.Second
var isWriting = false
var mu = sync.Mutex{}

func main() {
	fmt.Println("starting tcp client")
	server, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Err starting a tcp listener:", err)
		return
	}
	defer server.Close()

	fmt.Println("tcp listening")
	err = recoverInMemWall()
	if err != nil {
		fmt.Println("could not recover in mem wall:", err)
		os.Exit(1)
	}
	wallChan := make(chan []byte)
	disconnectChan := make(chan string)

	go writeToInMemWall(wallChan)
	go pruneConnection(disconnectChan)

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
		server.Close()
		os.Exit(0)
	}()

	go func() {
		for {
			boradcastForListeners(disconnectChan)
		}
	}()

	for {
		con, err := server.Accept()
		if err != nil {
			fmt.Println("Err accepting a connection: ", err)
			continue
		}
		clientConn := listener.NewListenerClient(con)
		clients = append(clients, clientConn)

		go handleClient(clientConn, wallChan, disconnectChan)
	}
}

func writeToInMemWall(wallChan chan []byte) error {
	for {
		select {
		case msgBts, ok := <-wallChan:
			if !ok {
				continue
			}

			message := listener.EventBusMessage{}
			err := json.Unmarshal(msgBts, &message)
			if err != nil {
				return err
			}
			message.Id = uuid.New()

			msgsInMem = append(msgsInMem, message)

		}
	}
}

func handleClient(listener *listener.ListenerClient, wallChan chan []byte, disconnectChan chan string) {
	defer listener.Close()
	err := listenerCatchUp(listener)
	if err != nil {
		fmt.Println("err listener catch up:", err)
		return
	}

	for {
		msg, err := listener.ReceiveMessage()
		if err != nil {
			fmt.Println("err reveiving message from listener:", err)
			disconnectChan <- listener.Id.String()
			return
		}

		log.Default().Printf("Got from conn: %s\n", string(msg))
		for _, client := range clients {
			if client.Id != listener.Id {
				client.AppendMessage(msg, wallChan)
			}
		}
	}
}

func boradcastForListeners(diconnectChan chan string) {
	for _, client := range clients {
		for client.CheckPendingMessages() != 0 {
			err := client.SendPendingMessage()
			if err != nil {
				fmt.Println("failed to send listener msg: ", err)
				diconnectChan <- client.Id.String()
				break
			}
		}
	}

}

func listenerCatchUp(client *listener.ListenerClient) error {
	for _, msg := range msgsInMem {
		if msg.SentAt.Add(deadTime).Unix() < time.Now().Unix() {
			continue
		}
		bts, err := json.Marshal(msg)
		if err != nil {
			log.Default().Println("failed to marshal catch up msg to client:", err)
			return err
		}

		client.AppendCatchUpMessage(bts)
	}

	return nil
}

func pruneConnection(disconnectChan chan string) {
	for {
		select {
		case id, ok := <-disconnectChan:
			if !ok {
				continue
			}
			mu.Lock()

			aliveClients := []*listener.ListenerClient{}
			for _, client := range clients {
				if client.Id.String() != id {
					aliveClients = append(aliveClients, client)
				}
			}

			clients = aliveClients
			mu.Unlock()
		}

	}
}

type wallMessages struct {
	Messages []listener.EventBusMessage `json:"messages"`
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

	removedDeadOnes := []listener.EventBusMessage{}
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

		contains := slices.ContainsFunc(removedDeadOnes, func(m listener.EventBusMessage) bool {
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
