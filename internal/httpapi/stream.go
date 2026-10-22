package httpapi

import (
	"bytes"
	"net/http"
	"time"
)

// stream serves the score server-sent-events stream. It costs one goroutine
// (the handler's) and never blocks writers: Broadcaster only keeps the latest
// message per subscriber, and every write has a deadline.
func (a *api) stream(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.session(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	// Subscribe before reading the first snapshot: a duplicate is harmless,
	// a missed change is not.
	sub := a.Broadcaster.Subscribe()
	defer a.Broadcaster.Unsubscribe(sub)

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)
	send := func(b []byte) bool {
		// Deadline errors (unsupported writers) are not fatal; Write errors are.
		_ = rc.SetWriteDeadline(time.Now().Add(a.StreamWriteTimeout))
		if _, err := w.Write(b); err != nil {
			return false
		}
		return rc.Flush() == nil
	}
	event := func(js []byte) bool {
		js = bytes.TrimRight(js, "\r\n")
		b := make([]byte, 0, len(js)+8)
		b = append(b, "data: "...)
		b = append(b, js...)
		b = append(b, '\n', '\n')
		return send(b)
	}

	if !event(a.State.ScoreboardJSON()) {
		return
	}
	ping := time.NewTicker(a.PingInterval)
	defer ping.Stop()
	expiry := time.NewTimer(sess.Expires.Sub(a.Now()))
	defer expiry.Stop()
	for {
		select {
		case <-sub.Ready():
			if m := sub.Take(); m != nil && !event(m) {
				return
			}
		case <-ping.C:
			if !send([]byte(": ping\n\n")) {
				return
			}
		case <-sess.Done:
			return
		case <-expiry.C:
			return
		case <-r.Context().Done():
			return
		}
	}
}
