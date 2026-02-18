// Package queue provides a thread-safe, bounded FIFO message queue with
// long-poll support for Discord message ingestion.
package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// QueuedMessage represents a single Discord message captured from a guild channel.
type QueuedMessage struct {
	ID               string    `json:"id"`
	ChannelID        string    `json:"channel_id"`
	ChannelName      string    `json:"channel_name"`
	AuthorID         string    `json:"author_id"`
	AuthorUsername   string    `json:"author_username"`
	Content          string    `json:"content"`
	Timestamp        time.Time `json:"timestamp"`
	MessageReference string    `json:"message_reference,omitempty"`
}

// Formatted returns a human-readable representation of the message in the
// form "[#channel] @user: text".
func (m QueuedMessage) Formatted() string {
	return fmt.Sprintf("[#%s] @%s: %s", m.ChannelName, m.AuthorUsername, m.Content)
}

// Option is a functional option for configuring a Queue.
type Option func(*Queue)

// WithMaxSize sets the maximum number of messages the queue can hold.
// Values of zero or less are ignored; the default of 1000 is used instead.
func WithMaxSize(n int) Option {
	return func(q *Queue) {
		if n > 0 {
			q.maxSize = n
		}
	}
}

// Queue is a thread-safe, bounded FIFO ring-buffer queue. When the buffer is
// full, the oldest message is silently dropped to make room for the new one.
// Callers waiting in Poll are notified via a broadcast channel whenever a new
// message is enqueued.
type Queue struct {
	mu      sync.Mutex
	buf     []QueuedMessage
	head    int
	count   int
	maxSize int
	notify  chan struct{}
}

// New constructs a Queue with the provided options applied. The default
// maximum size is 1000 messages.
func New(opts ...Option) *Queue {
	q := &Queue{
		maxSize: 1000,
		notify:  make(chan struct{}),
	}
	for _, opt := range opts {
		opt(q)
	}
	q.buf = make([]QueuedMessage, q.maxSize)
	return q
}

// Enqueue adds msg to the tail of the queue. If the queue is full, the oldest
// message (at head) is discarded to accommodate the new one. Enqueue never
// blocks and wakes all goroutines currently blocked in Poll.
func (q *Queue) Enqueue(msg QueuedMessage) {
	q.mu.Lock()

	if q.count == q.maxSize {
		// Drop the oldest message by advancing head.
		q.head = (q.head + 1) % q.maxSize
		q.count--
	}

	tail := (q.head + q.count) % q.maxSize
	q.buf[tail] = msg
	q.count++

	// Broadcast to all waiters: close the old channel and replace it.
	oldNotify := q.notify
	q.notify = make(chan struct{})

	q.mu.Unlock()

	close(oldNotify)
}

// poll collects up to limit messages from the queue into dst, applying an
// optional channelFilter. When channelFilter is non-empty only messages whose
// ChannelID or ChannelName matches it are returned; non-matching messages
// remain in the ring buffer. The caller must hold q.mu.
func (q *Queue) poll(channelFilter string, limit int) []QueuedMessage {
	if q.count == 0 {
		return nil
	}

	if channelFilter == "" {
		// Fast path: collect up to limit messages from the head.
		n := q.count
		if limit > 0 && n > limit {
			n = limit
		}
		out := make([]QueuedMessage, n)
		for i := 0; i < n; i++ {
			idx := (q.head + i) % q.maxSize
			out[i] = q.buf[idx]
			q.buf[idx] = QueuedMessage{}
		}
		q.head = (q.head + n) % q.maxSize
		q.count -= n
		return out
	}

	// Filtered path: scan all messages, collect matching ones, compact buffer.
	var out []QueuedMessage
	kept := make([]QueuedMessage, 0, q.count)

	for i := 0; i < q.count; i++ {
		msg := q.buf[(q.head+i)%q.maxSize]
		collected := limit <= 0 || len(out) < limit
		if collected && (msg.ChannelID == channelFilter || msg.ChannelName == channelFilter) {
			out = append(out, msg)
		} else {
			kept = append(kept, msg)
		}
	}

	// Rewrite the ring buffer with only the kept messages.
	q.head = 0
	q.count = len(kept)
	copy(q.buf, kept)
	// Zero out trailing slots to release stale references.
	for i := len(kept); i < q.maxSize; i++ {
		q.buf[i] = QueuedMessage{}
	}

	return out
}

// Poll returns up to limit messages from the queue, blocking until at least
// one message is available, the timeout expires, or ctx is cancelled.
//
// When channelFilter is non-empty only messages whose ChannelID or ChannelName
// equals channelFilter are returned; messages that do not match are left in the
// buffer for future calls.
//
// A limit of zero or less means return all available matching messages.
// Messages are returned in FIFO order (oldest first) and are removed from the
// queue; each message is delivered at most once.
//
// Poll returns nil (not an error) when the timeout elapses or ctx is cancelled
// with no messages to deliver.
func (q *Queue) Poll(ctx context.Context, timeout time.Duration, limit int, channelFilter string) []QueuedMessage {
	// Try immediately first.
	q.mu.Lock()
	if msgs := q.poll(channelFilter, limit); len(msgs) > 0 {
		q.mu.Unlock()
		return msgs
	}
	// Capture the current notify channel while still holding the lock so we
	// don't miss a signal that arrives between the lock release and the select.
	notifyCh := q.notify
	q.mu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			return nil
		case <-notifyCh:
			// A message was enqueued; try to collect.
			q.mu.Lock()
			msgs := q.poll(channelFilter, limit)
			notifyCh = q.notify
			q.mu.Unlock()
			if len(msgs) > 0 {
				return msgs
			}
			// The message may not have matched our filter; keep waiting.
		}
	}
}

// Len returns the current number of messages in the queue.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.count
}
