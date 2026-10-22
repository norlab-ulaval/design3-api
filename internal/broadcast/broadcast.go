// Package broadcast fans the latest message out to subscribers without ever
// blocking the publisher. Slow subscribers only ever see the newest value.
package broadcast

import "sync"

// Broadcaster delivers published messages to all current subscribers.
type Broadcaster struct {
	mu   sync.Mutex
	subs map[*Sub]struct{}
}

// Sub is one subscriber holding at most one pending (latest) message.
type Sub struct {
	mu      sync.Mutex
	pending []byte
	ready   chan struct{}
}

// New returns an empty Broadcaster.
func New() *Broadcaster {
	return &Broadcaster{subs: make(map[*Sub]struct{})}
}

// Subscribe registers and returns a new subscriber.
func (b *Broadcaster) Subscribe() *Sub {
	s := &Sub{ready: make(chan struct{}, 1)}
	b.mu.Lock()
	b.subs[s] = struct{}{}
	b.mu.Unlock()
	return s
}

// Unsubscribe removes s; it receives nothing further.
func (b *Broadcaster) Unsubscribe(s *Sub) {
	b.mu.Lock()
	delete(b.subs, s)
	b.mu.Unlock()
}

// Count returns the number of current subscribers.
func (b *Broadcaster) Count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs)
}

// Publish replaces every subscriber's pending value with msg. It never blocks
// on a subscriber. msg must not be modified by the caller afterwards.
func (b *Broadcaster) Publish(msg []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for s := range b.subs {
		s.mu.Lock()
		s.pending = msg
		s.mu.Unlock()
		select {
		case s.ready <- struct{}{}:
		default:
		}
	}
}

// Ready is signalled (1-buffered) when a value is pending.
func (s *Sub) Ready() <-chan struct{} { return s.ready }

// Take returns and clears the pending value, or nil if none.
func (s *Sub) Take() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.pending
	s.pending = nil
	return m
}
