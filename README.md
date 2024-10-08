# Event Bus

This is a little project for learning how event busses work written in Go, written from scratch. Still a work in progess, need to clean up the bus code a bit to make it easier to configure.

## Bus
The bus is a TCP server that takes in listener connections, saves, manages them and forwards events
to other listeners. In order to limit event spam so that the listeners wont get malformatted messages, each
listeners has its own queue that new messages get appended to. The server then concurrently forwards the messages
from the queues to the listneres 1 by 1.

Managing connections is done via a go channel, whenever sending or receiving a message from client errors, the connections id gets pushed to a channel after which the server drops and cleans the connection from saved connections.

The server also keeps a log of most recent events in a wall that periodically gets saved on the disk. When a new client
connects to the server, they get caught up with the most recent messages from the wall.

## Client
Most of the client code is currently for testing message sends and receiving via topics. However, it also includes the logic for abstracting away connecting to teh bus, message receiving and sending into an easy to use interface. 
