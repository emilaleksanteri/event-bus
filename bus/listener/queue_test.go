package listener

import "testing"

func TestQueueInsert(t *testing.T) {
	queue := Queue{}
	queue.Append(Node{})
	if queue.Size != 1 {
		t.Fatalf("queue was not size 1, got %d", queue.Size)
	}

	queue.Append(Node{})
	if queue.Size != 2 {
		t.Fatalf("queue was not size 2, got %d", queue.Size)
	}
}

func TestQueuePop(t *testing.T) {
	queue := Queue{}
	queue.Append(Node{Data: []byte("hi")})
	queue.Append(Node{Data: []byte("hi2")})

	res := queue.Pop()
	if string(res.Data) != "hi" {
		t.Fatalf("did not get hi from pop, got %s", string(res.Data))
	}
	if queue.Size != 1 {
		t.Fatalf("after popping queue should have been size 1, got %d", queue.Size)
	}

	res = queue.Pop()
	if string(res.Data) != "hi2" {
		t.Fatalf("did not get hi2 from pop, got %s", string(res.Data))
	}

	if queue.Size != 0 {
		t.Fatalf("after popping queue should have been size 0, got %d", queue.Size)
	}

}
