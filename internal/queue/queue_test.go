package queue

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// New
// ---------------------------------------------------------------------------

func Test_New_NoOptions(t *testing.T) {
	t.Parallel()
	q := New()
	if q == nil {
		t.Fatal("New() returned nil")
	}
	if q.Len() != 0 {
		t.Errorf("New().Len() = %d, want 0", q.Len())
	}
}

func Test_New_WithMaxSize(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(10))
	if q == nil {
		t.Fatal("New(WithMaxSize(10)) returned nil")
	}
	if q.Len() != 0 {
		t.Errorf("New(WithMaxSize(10)).Len() = %d, want 0", q.Len())
	}
}

func Test_New_WithMaxSize_Zero_FallsBack(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(0))
	if q == nil {
		t.Fatal("New(WithMaxSize(0)) returned nil")
	}
	// The queue should fall back to 1000 max size. We verify by filling
	// up to 1000 and checking nothing is dropped prematurely.
	for i := 0; i < 1000; i++ {
		q.Enqueue(QueuedMessage{Content: fmt.Sprintf("msg-%d", i)})
	}
	if q.Len() != 1000 {
		t.Errorf("After filling 1000 messages, Len() = %d, want 1000", q.Len())
	}
	// Adding one more should drop the oldest (proving max is 1000, not unlimited).
	q.Enqueue(QueuedMessage{Content: "overflow"})
	if q.Len() != 1000 {
		t.Errorf("After overflow, Len() = %d, want 1000", q.Len())
	}
}

func Test_New_WithMaxSize_Negative_FallsBack(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(-1))
	if q == nil {
		t.Fatal("New(WithMaxSize(-1)) returned nil")
	}
	// Same check as zero: fallback to 1000.
	for i := 0; i < 1000; i++ {
		q.Enqueue(QueuedMessage{Content: fmt.Sprintf("msg-%d", i)})
	}
	if q.Len() != 1000 {
		t.Errorf("After filling 1000 messages, Len() = %d, want 1000", q.Len())
	}
}

// ---------------------------------------------------------------------------
// Enqueue
// ---------------------------------------------------------------------------

func Test_Enqueue_ToEmpty(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(3))
	q.Enqueue(QueuedMessage{Content: "a"})
	if q.Len() != 1 {
		t.Errorf("Len() = %d after enqueue to empty queue, want 1", q.Len())
	}
}

func Test_Enqueue_NotFull(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(3))
	q.Enqueue(QueuedMessage{Content: "a"})
	q.Enqueue(QueuedMessage{Content: "b"})
	q.Enqueue(QueuedMessage{Content: "c"})
	if q.Len() != 3 {
		t.Errorf("Len() = %d after 3 enqueues, want 3", q.Len())
	}
}

func Test_Enqueue_Full_DropsOldest(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(3))
	q.Enqueue(QueuedMessage{Content: "a"})
	q.Enqueue(QueuedMessage{Content: "b"})
	q.Enqueue(QueuedMessage{Content: "c"})

	// Queue is full (3/3). Enqueue one more — oldest should be dropped.
	q.Enqueue(QueuedMessage{Content: "d"})
	if q.Len() != 3 {
		t.Errorf("Len() = %d after overflow enqueue, want 3", q.Len())
	}

	// Poll all — should get b, c, d (a was dropped).
	ctx := context.Background()
	msgs := q.Poll(ctx, time.Second, 10, "")
	if len(msgs) != 3 {
		t.Fatalf("Poll returned %d messages, want 3", len(msgs))
	}
	wantContents := []string{"b", "c", "d"}
	for i, want := range wantContents {
		if msgs[i].Content != want {
			t.Errorf("msgs[%d].Content = %q, want %q", i, msgs[i].Content, want)
		}
	}
}

func Test_Enqueue_Full_BulkOverflow(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(3))
	q.Enqueue(QueuedMessage{Content: "a"})
	q.Enqueue(QueuedMessage{Content: "b"})
	q.Enqueue(QueuedMessage{Content: "c"})

	// Enqueue 4 more on a full(3) queue — only the last 3 should remain.
	q.Enqueue(QueuedMessage{Content: "d"})
	q.Enqueue(QueuedMessage{Content: "e"})
	q.Enqueue(QueuedMessage{Content: "f"})
	q.Enqueue(QueuedMessage{Content: "g"})
	if q.Len() != 3 {
		t.Errorf("Len() = %d after bulk overflow, want 3", q.Len())
	}

	ctx := context.Background()
	msgs := q.Poll(ctx, time.Second, 10, "")
	if len(msgs) != 3 {
		t.Fatalf("Poll returned %d messages, want 3", len(msgs))
	}
	wantContents := []string{"e", "f", "g"}
	for i, want := range wantContents {
		if msgs[i].Content != want {
			t.Errorf("msgs[%d].Content = %q, want %q", i, msgs[i].Content, want)
		}
	}
}

func Test_Enqueue_Concurrent(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(50))

	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(n int) {
			defer wg.Done()
			q.Enqueue(QueuedMessage{Content: fmt.Sprintf("msg-%d", n)})
		}(i)
	}
	wg.Wait()

	if q.Len() > 50 {
		t.Errorf("Len() = %d after 100 concurrent enqueues with maxSize=50, want <= 50", q.Len())
	}
	if q.Len() == 0 {
		t.Error("Len() = 0 after 100 concurrent enqueues, expected > 0")
	}
}

// ---------------------------------------------------------------------------
// Poll
// ---------------------------------------------------------------------------

func Test_Poll_ReturnsAllImmediately(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(10))
	q.Enqueue(QueuedMessage{Content: "a", ChannelName: "gen"})
	q.Enqueue(QueuedMessage{Content: "b", ChannelName: "gen"})
	q.Enqueue(QueuedMessage{Content: "c", ChannelName: "gen"})

	ctx := context.Background()
	msgs := q.Poll(ctx, time.Second, 10, "")
	if len(msgs) != 3 {
		t.Fatalf("Poll returned %d messages, want 3", len(msgs))
	}
}

func Test_Poll_LimitRespected(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(10))
	q.Enqueue(QueuedMessage{Content: "a"})
	q.Enqueue(QueuedMessage{Content: "b"})
	q.Enqueue(QueuedMessage{Content: "c"})

	ctx := context.Background()
	msgs := q.Poll(ctx, time.Second, 2, "")
	if len(msgs) != 2 {
		t.Fatalf("Poll(limit=2) returned %d messages, want 2", len(msgs))
	}
	// Should return oldest first (FIFO).
	if msgs[0].Content != "a" {
		t.Errorf("msgs[0].Content = %q, want %q", msgs[0].Content, "a")
	}
	if msgs[1].Content != "b" {
		t.Errorf("msgs[1].Content = %q, want %q", msgs[1].Content, "b")
	}
}

func Test_Poll_ChannelFilter(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(10))
	q.Enqueue(QueuedMessage{Content: "a1", ChannelName: "a"})
	q.Enqueue(QueuedMessage{Content: "b1", ChannelName: "b"})
	q.Enqueue(QueuedMessage{Content: "a2", ChannelName: "a"})

	ctx := context.Background()
	msgs := q.Poll(ctx, time.Second, 10, "a")
	if len(msgs) != 2 {
		t.Fatalf("Poll(channel='a') returned %d messages, want 2", len(msgs))
	}
	if msgs[0].Content != "a1" {
		t.Errorf("msgs[0].Content = %q, want %q", msgs[0].Content, "a1")
	}
	if msgs[1].Content != "a2" {
		t.Errorf("msgs[1].Content = %q, want %q", msgs[1].Content, "a2")
	}
}

func Test_Poll_EmptyQueue_WaitsForTimeout(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(10))

	ctx := context.Background()
	start := time.Now()
	msgs := q.Poll(ctx, 100*time.Millisecond, 10, "")
	elapsed := time.Since(start)

	if len(msgs) != 0 {
		t.Errorf("Poll on empty queue returned %d messages, want 0", len(msgs))
	}
	// Should have waited approximately 100ms.
	if elapsed < 50*time.Millisecond {
		t.Errorf("Poll returned too quickly: %v, expected at least ~100ms", elapsed)
	}
}

func Test_Poll_WakeUpOnEnqueue(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(10))

	ctx := context.Background()

	// Enqueue a message after a short delay while Poll is waiting.
	go func() {
		time.Sleep(50 * time.Millisecond)
		q.Enqueue(QueuedMessage{Content: "wakeup"})
	}()

	start := time.Now()
	msgs := q.Poll(ctx, 5*time.Second, 10, "")
	elapsed := time.Since(start)

	if len(msgs) != 1 {
		t.Fatalf("Poll returned %d messages, want 1", len(msgs))
	}
	if msgs[0].Content != "wakeup" {
		t.Errorf("msgs[0].Content = %q, want %q", msgs[0].Content, "wakeup")
	}
	// Should not have waited the full 5s timeout.
	if elapsed > 2*time.Second {
		t.Errorf("Poll took %v, expected to wake up well before 5s timeout", elapsed)
	}
}

func Test_Poll_ChannelFilter_NoMatch(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(10))
	q.Enqueue(QueuedMessage{Content: "a1", ChannelName: "a"})
	q.Enqueue(QueuedMessage{Content: "b1", ChannelName: "b"})

	ctx := context.Background()
	msgs := q.Poll(ctx, 100*time.Millisecond, 10, "c")
	if len(msgs) != 0 {
		t.Errorf("Poll(channel='c') returned %d messages, want 0 (no matches)", len(msgs))
	}
}

func Test_Poll_RemovesReturnedMessages(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(10))
	q.Enqueue(QueuedMessage{Content: "a"})
	q.Enqueue(QueuedMessage{Content: "b"})

	ctx := context.Background()
	msgs := q.Poll(ctx, time.Second, 10, "")
	if len(msgs) != 2 {
		t.Fatalf("First Poll returned %d messages, want 2", len(msgs))
	}

	// Queue should now be empty.
	if q.Len() != 0 {
		t.Errorf("Len() = %d after draining poll, want 0", q.Len())
	}
}

func Test_Poll_NonMatchingRemain(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(10))
	q.Enqueue(QueuedMessage{Content: "a1", ChannelName: "a"})
	q.Enqueue(QueuedMessage{Content: "b1", ChannelName: "b"})
	q.Enqueue(QueuedMessage{Content: "a2", ChannelName: "a"})

	ctx := context.Background()
	msgs := q.Poll(ctx, time.Second, 10, "a")
	if len(msgs) != 2 {
		t.Fatalf("Poll(channel='a') returned %d messages, want 2", len(msgs))
	}

	// "b1" should still be in the queue.
	if q.Len() != 1 {
		t.Errorf("Len() = %d after filtered poll, want 1 (non-matching 'b' should remain)", q.Len())
	}
}

func Test_Poll_CancelledContext(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(10))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	start := time.Now()
	msgs := q.Poll(ctx, 5*time.Second, 10, "")
	elapsed := time.Since(start)

	if len(msgs) != 0 {
		t.Errorf("Poll with cancelled context returned %d messages, want 0", len(msgs))
	}
	if elapsed > time.Second {
		t.Errorf("Poll with cancelled context took %v, expected immediate return", elapsed)
	}
}

func Test_Poll_FIFO_Order(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(10))
	for i := 0; i < 5; i++ {
		q.Enqueue(QueuedMessage{Content: fmt.Sprintf("msg-%d", i)})
	}

	ctx := context.Background()
	msgs := q.Poll(ctx, time.Second, 10, "")
	if len(msgs) != 5 {
		t.Fatalf("Poll returned %d messages, want 5", len(msgs))
	}
	for i, msg := range msgs {
		want := fmt.Sprintf("msg-%d", i)
		if msg.Content != want {
			t.Errorf("msgs[%d].Content = %q, want %q (FIFO order)", i, msg.Content, want)
		}
	}
}

func Test_Poll_Concurrent_NoDuplicates(t *testing.T) {
	t.Parallel()
	q := New(WithMaxSize(100))
	for i := 0; i < 50; i++ {
		q.Enqueue(QueuedMessage{Content: fmt.Sprintf("msg-%d", i)})
	}

	ctx := context.Background()
	var mu sync.Mutex
	allReceived := make([]QueuedMessage, 0, 50)

	var wg sync.WaitGroup
	numPollers := 10
	wg.Add(numPollers)
	for i := 0; i < numPollers; i++ {
		go func() {
			defer wg.Done()
			msgs := q.Poll(ctx, 200*time.Millisecond, 10, "")
			mu.Lock()
			allReceived = append(allReceived, msgs...)
			mu.Unlock()
		}()
	}
	wg.Wait()

	// Check no duplicate messages were received.
	seen := make(map[string]bool)
	for _, msg := range allReceived {
		if seen[msg.Content] {
			t.Errorf("duplicate message received: %q", msg.Content)
		}
		seen[msg.Content] = true
	}

	// All 50 messages should eventually be consumed.
	if len(allReceived) != 50 {
		t.Errorf("total messages received = %d, want 50", len(allReceived))
	}
}

// ---------------------------------------------------------------------------
// QueuedMessage.Formatted
// ---------------------------------------------------------------------------

func Test_QueuedMessage_Formatted_Cases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  QueuedMessage
		want string
	}{
		{
			name: "all fields populated",
			msg: QueuedMessage{
				ChannelName:    "general",
				AuthorUsername: "alice",
				Content:        "hello",
			},
			want: "[#general] @alice: hello",
		},
		{
			name: "empty channel name",
			msg: QueuedMessage{
				ChannelName:    "",
				AuthorUsername: "bob",
				Content:        "test",
			},
			want: "[#] @bob: test",
		},
		{
			name: "empty author",
			msg: QueuedMessage{
				ChannelName:    "general",
				AuthorUsername: "",
				Content:        "msg",
			},
			want: "[#general] @: msg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.msg.Formatted()
			if got != tt.want {
				t.Errorf("Formatted() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func Benchmark_Enqueue_NonFull(b *testing.B) {
	q := New(WithMaxSize(b.N + 1))
	msg := QueuedMessage{Content: "bench", ChannelName: "gen", AuthorUsername: "user"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Enqueue(msg)
	}
}

func Benchmark_Poll_SingleMessage(b *testing.B) {
	ctx := context.Background()
	q := New(WithMaxSize(b.N + 1))
	msg := QueuedMessage{Content: "bench", ChannelName: "gen", AuthorUsername: "user"}
	for i := 0; i < b.N; i++ {
		q.Enqueue(msg)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Poll(ctx, time.Millisecond, 1, "")
	}
}

func Benchmark_QueuedMessage_Formatted(b *testing.B) {
	msg := QueuedMessage{
		ChannelName:    "general",
		AuthorUsername: "alice",
		Content:        "hello world this is a benchmark message",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = msg.Formatted()
	}
}
