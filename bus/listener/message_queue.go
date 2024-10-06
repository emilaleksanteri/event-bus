package listener

import "sync"

type Queue struct {
	Size int
	head *Node
	mu   sync.Mutex
}

type Node struct {
	next *Node
	Data []byte
}

func (q *Queue) Append(node Node) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.Size == 0 {
		q.Size += 1
		q.head = &node
		return
	}

	q.Size += 1

	curr := q.head
	for curr.next != nil {
		curr = curr.next
	}

	curr.next = &node
}

func (q *Queue) Pop() *Node {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.Size == 0 {
		return nil
	}
	q.Size -= 1

	head := q.head
	q.head = head.next

	return head
}
