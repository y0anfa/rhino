package queue

import (
	"context"
	"testing"
	"time"
)

func TestMemoryQueue_EnqueueDequeue(t *testing.T) {
	q := NewMemoryQueue()
	defer q.Close()

	msg := &TaskMessage{ID: "t1", TaskName: "task1", EnqueuedAt: time.Now()}
	if err := q.Enqueue(msg); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	if q.Len() != 1 {
		t.Errorf("expected len=1, got %d", q.Len())
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}
	if got.ID != "t1" {
		t.Errorf("expected id=t1, got %s", got.ID)
	}
}

func TestMemoryQueue_FIFO(t *testing.T) {
	q := NewMemoryQueue()
	defer q.Close()

	q.Enqueue(&TaskMessage{ID: "a"})
	q.Enqueue(&TaskMessage{ID: "b"})
	q.Enqueue(&TaskMessage{ID: "c"})

	ctx := context.Background()
	m1, _ := q.Dequeue(ctx)
	m2, _ := q.Dequeue(ctx)
	m3, _ := q.Dequeue(ctx)

	if m1.ID != "a" || m2.ID != "b" || m3.ID != "c" {
		t.Errorf("expected FIFO order a,b,c got %s,%s,%s", m1.ID, m2.ID, m3.ID)
	}
}

func TestMemoryQueue_AckNack(t *testing.T) {
	q := NewMemoryQueue()
	defer q.Close()

	q.Enqueue(&TaskMessage{ID: "t1"})

	ctx := context.Background()
	msg, _ := q.Dequeue(ctx)

	// Nack re-enqueues
	q.Nack(msg.ID)
	if q.Len() != 1 {
		t.Errorf("expected len=1 after nack, got %d", q.Len())
	}

	msg, _ = q.Dequeue(ctx)
	q.Ack(msg.ID)
	if q.Len() != 0 {
		t.Errorf("expected len=0 after ack, got %d", q.Len())
	}
}

func TestMemoryQueue_DequeueBlocks(t *testing.T) {
	q := NewMemoryQueue()
	defer q.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := q.Dequeue(ctx)
	if err == nil {
		t.Error("expected timeout error on empty queue")
	}
}

func TestMemoryQueue_DequeueUnblocksOnEnqueue(t *testing.T) {
	q := NewMemoryQueue()
	defer q.Close()

	go func() {
		time.Sleep(20 * time.Millisecond)
		q.Enqueue(&TaskMessage{ID: "late"})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	msg, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}
	if msg.ID != "late" {
		t.Errorf("expected id=late, got %s", msg.ID)
	}
}
