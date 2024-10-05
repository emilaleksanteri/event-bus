package main

import (
	"fmt"
	"net"

	"github.com/google/uuid"
)

var clients = []*listenerClient{}

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

func main() {
	fmt.Println("starting tcp client")
	listener, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Err starting a tcp listener:", err)
		return
	}

	fmt.Println("tcp listening")

	defer listener.Close()
	for {
		con, err := listener.Accept()
		if err != nil {
			fmt.Println("Err accepting a connection: ", err)
			continue
		}
		clientConn := newListenerClient(con)
		clients = append(clients, clientConn)

		go handleClient(clientConn)
	}
}

func handleClient(listener *listenerClient) {
	defer listener.conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, err := listener.conn.Read(buffer)
		if err != nil {
			fmt.Println("err reading from tcp conn:", err)
			return
		}

		fmt.Printf("Got from conn: %s\n", buffer[:n])
		for _, client := range clients {
			if client.id != listener.id {
				_, err := client.conn.Write(buffer[:n])
				if err != nil {
					fmt.Println("err broadcasting", err)
				}
			}
		}
	}
}
