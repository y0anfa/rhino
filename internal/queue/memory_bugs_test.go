package queue

import (
	"context"
	"testing"
	"time"
)

func TestMemoryQueue_CloseIsIdempotent(t *testing.T) {
	q := NewMemoryQueue()
	if err := q.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	if err := q.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}
}

// Every queued message must reach a waiting consumer, even when several are
// enqueued before any of them wakes up.
func TestMemoryQueue_WakesWaiterPerMessage(t *testing.T) {
	q := NewMemoryQueue()
	defer q.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got := make(chan string, 2)
	for i := 0; i < 2; i++ {
		go func() {
			msg, err := q.Dequeue(ctx)
			if err == nil {
				got <- msg.ID
			}
		}()
	}

	time.Sleep(100 * time.Millisecond) // let both consumers block on the queue

	if err := q.Enqueue(&TaskMessage{ID: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(&TaskMessage{ID: "b"}); err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case id := <-got:
			seen[id] = true
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d of 2 messages were delivered (%v)", len(seen), seen)
		}
	}
	if !seen["a"] || !seen["b"] {
		t.Errorf("expected both messages, got %v", seen)
	}
}
