package broadcast

import (
	"sync"
	"testing"
	"time"
)

func TestLatestOnly(t *testing.T) {
	b := New()
	s := b.Subscribe()
	b.Publish([]byte("a"))
	b.Publish([]byte("b"))
	b.Publish([]byte("c"))
	<-s.Ready()
	if got := string(s.Take()); got != "c" {
		t.Fatalf("got %q, want c", got)
	}
	select {
	case <-s.Ready():
		t.Fatal("unexpected extra signal")
	default:
	}
	if s.Take() != nil {
		t.Fatal("expected nil after Take")
	}
}

func TestPublishNeverBlocks(t *testing.T) {
	b := New()
	for i := 0; i < 1000; i++ {
		b.Subscribe()
	}
	start := time.Now()
	for i := 0; i < 10000; i++ {
		b.Publish([]byte("x"))
	}
	limit := time.Second
	if raceEnabled {
		limit = 10 * time.Second // the race detector slows this loop ~15x
	}
	if d := time.Since(start); d > limit {
		t.Fatalf("publish took %v (limit %v)", d, limit)
	}
}

func TestUnsubscribe(t *testing.T) {
	b := New()
	s1 := b.Subscribe()
	s2 := b.Subscribe()
	if b.Count() != 2 {
		t.Fatalf("count %d", b.Count())
	}
	b.Unsubscribe(s1)
	if b.Count() != 1 {
		t.Fatalf("count %d", b.Count())
	}
	b.Publish([]byte("new"))
	select {
	case <-s1.Ready():
		t.Fatal("unsubscribed sub signalled")
	default:
	}
	if s1.Take() != nil {
		t.Fatal("unsubscribed sub got value")
	}
	<-s2.Ready()
	if string(s2.Take()) != "new" {
		t.Fatal("s2 missing value")
	}
}

func TestConcurrent(t *testing.T) {
	b := New()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				b.Publish([]byte("m"))
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				s := b.Subscribe()
				select {
				case <-s.Ready():
					s.Take()
				default:
				}
				b.Unsubscribe(s)
				b.Count()
			}
		}()
	}
	wg.Wait()
	if b.Count() != 0 {
		t.Fatalf("count %d", b.Count())
	}
}
