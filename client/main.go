package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	go newClient("emil 1")
	go newClient("emil 2")
	for {
	}
}

func newClient(name string) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Err dialing tcp: ", err)
		return
	}
	defer conn.Close()

	time.Sleep(1 * time.Second)
	data := []byte(name + " said: Hello there")
	_, err = conn.Write(data)
	if err != nil {
		fmt.Println("err sending data over tcp:", err)
		return
	}

	for {

		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("err reading:", err)
			continue
		}

		fmt.Printf("%s got a broadcast: %s\n", name, buffer[:n])
	}

}
