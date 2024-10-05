package main

import (
	"encoding/json"
	"fmt"
	"net"
	"slices"
	"time"
)

var topic = "my-topic"

func main() {
	client1, err := NewClient("emil1", []string{topic})
	if err != nil {
		fmt.Println("could not make client:", err)
		return
	}
	client2, err := NewClient("emil2", []string{topic})
	if err != nil {
		fmt.Println("could not make client", err)
		return
	}

	go testClientReceive(client1)
	go testClientReceive(client2)
	go testclientSend(client1)
	go testclientSend(client2)
	for {
	}
}

func testClientReceive(client *EventBusClient) {
	for {
		msg, err := client.Receive()
		if err != nil {
			fmt.Println("err receiving on name:", client.name)
		}

		fmt.Printf("msg: %+v\n", msg)
	}
}

func testclientSend(client *EventBusClient) {
	for {
		time.Sleep(2 * time.Second)
		msg := EventBusMessage{
			Topic:  topic,
			Body:   "hello from " + client.name,
			SentAt: time.Now(),
		}
		err := client.Write(msg)
		if err != nil {
			fmt.Println("err sending msg: ", err, "client: ", client.name)
		}
	}
}

type EventBusMessage struct {
	Topic  string    `json:"topic"`
	Body   string    `json:"body"`
	SentAt time.Time `json:"sent_at"`
}

type EventBusClient struct {
	name         string
	conn         net.Conn
	subscribedTo []string
}

func (ec *EventBusClient) Write(msg EventBusMessage) error {
	inJson, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = ec.conn.Write(inJson)
	if err != nil {
		return err
	}

	return nil
}

func (ec *EventBusClient) Close() error {
	return ec.conn.Close()
}

func (ec *EventBusClient) subscribeToEvents(events []string) {
	ec.subscribedTo = append(ec.subscribedTo, events...)
}

func (ec *EventBusClient) Receive() (EventBusMessage, error) {
	message := EventBusMessage{}

	for {
		buffer := make([]byte, 1024)
		n, err := ec.conn.Read(buffer)
		if err != nil {
			return EventBusMessage{}, err
		}

		bts := buffer[:n]
		if string(bts) == "ping" {
			continue
		}

		msg := EventBusMessage{}
		err = json.Unmarshal(bts, &msg)
		if err != nil {
			return msg, err
		}

		if slices.Contains(ec.subscribedTo, msg.Topic) {
			message = msg
			break
		}
	}

	return message, nil
}

func NewClient(name string, topics []string) (*EventBusClient, error) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		return nil, err
	}

	client := &EventBusClient{
		name:         name,
		conn:         conn,
		subscribedTo: topics,
	}

	return client, nil
}
