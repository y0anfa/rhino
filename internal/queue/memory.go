package queue

import (
	"context"
	"fmt"
	"sync"
)

type MemoryQueue struct {
	items   []*TaskMessage
	pending map[string]*TaskMessage // id -> message (dequeued but not acked)
	mu      sync.Mutex
	notify  chan struct{}
	closed  bool
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		pending: make(map[string]*TaskMessage),
		notify:  make(chan struct{}, 1),
	}
}

func (q *MemoryQueue) Enqueue(msg *TaskMessage) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return fmt.Errorf("queue is closed")
	}
	q.items = append(q.items, msg)
	q.signal()
	return nil
}

// signal wakes one waiting consumer. The caller must hold q.mu.
func (q *MemoryQueue) signal() {
	if q.closed {
		return
	}
	// Non-blocking signal
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

func (q *MemoryQueue) Dequeue(ctx context.Context) (*TaskMessage, error) {
	for {
		q.mu.Lock()
		if len(q.items) > 0 {
			msg := q.items[0]
			q.items = q.items[1:]
			q.pending[msg.ID] = msg
			if len(q.items) > 0 {
				// Signals are coalesced, so hand the wake-up on to the next consumer.
				q.signal()
			}
			q.mu.Unlock()
			return msg, nil
		}
		if q.closed {
			q.mu.Unlock()
			return nil, fmt.Errorf("queue is closed")
		}
		q.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-q.notify:
		}
	}
}

func (q *MemoryQueue) Ack(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.pending, id)
	return nil
}

func (q *MemoryQueue) Nack(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if msg, ok := q.pending[id]; ok {
		delete(q.pending, id)
		q.items = append(q.items, msg) // re-enqueue
		q.signal()
	}
	return nil
}

func (q *MemoryQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func (q *MemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return nil
	}
	q.closed = true
	close(q.notify)
	return nil
}
