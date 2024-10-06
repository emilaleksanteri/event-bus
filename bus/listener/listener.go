package listener

import (
	"bufio"
	"fmt"
	"net"

	"github.com/google/uuid"
)

type ListenerClient struct {
	conn            net.Conn
	Id              uuid.UUID
	pendingMessages Queue
	sendingMessage  bool
}

func NewListenerClient(conn net.Conn) *ListenerClient {
	return &ListenerClient{
		conn:            conn,
		Id:              uuid.New(),
		pendingMessages: Queue{Size: 0},
	}
}

func (lc *ListenerClient) SendPendingMessage() error {
	for lc.sendingMessage {
	}
	lc.sendingMessage = true
	defer func() { lc.sendingMessage = false }()

	msg := lc.pendingMessages.Pop()
	if msg == nil {
		return nil
	}

	fmt.Println("sending", string(msg.Data))

	msg.Data = append(msg.Data, byte('\n'))
	_, err := lc.conn.Write(msg.Data)
	if err != nil {
		return err
	}

	return nil
}

func (lc *ListenerClient) AppendMessage(message []byte, wallChan chan []byte) {
	lc.pendingMessages.Append(Node{Data: message})
	wallChan <- message
}

func (lc *ListenerClient) AppendCatchUpMessage(message []byte) {
	lc.pendingMessages.Append(Node{Data: message})
}

func (lc *ListenerClient) Close() error {
	return lc.conn.Close()
}

func (lc *ListenerClient) CheckPendingMessages() int {
	return lc.pendingMessages.Size
}

func (lc *ListenerClient) ReceiveMessage() ([]byte, error) {
	bts, err := bufio.NewReader(lc.conn).ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	return bts, nil
}
