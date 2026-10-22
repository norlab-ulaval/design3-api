package httpapi

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"design3api/internal/auth"
)

// openStream opens /scores/stream with the client and returns a line channel
// that is closed at EOF.
func openStream(t *testing.T, c *http.Client, a *testAPI) (*http.Response, <-chan string) {
	t.Helper()
	resp, err := c.Get(a.URL + "/scores/stream")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	lines := make(chan string, 1024)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()
	return resp, lines
}

func nextData(t *testing.T, lines <-chan string) string {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case l, ok := <-lines:
			if !ok {
				t.Fatal("stream ended")
			}
			if strings.HasPrefix(l, "data: ") {
				return strings.TrimPrefix(l, "data: ")
			}
		case <-timeout:
			t.Fatal("timeout waiting for data")
		}
	}
}

func TestStreamRequiresSession(t *testing.T) {
	a := newTestAPI(t)
	s, _, b := a.do(t, "GET", "/scores/stream", "", "")
	if s != 401 || !strings.Contains(b, "Unauthorized") {
		t.Errorf("%d %s", s, b)
	}
}

func allZero(s string) bool { return !strings.Contains(s, `"points":1`) }

func TestStreamInitialAndChanges(t *testing.T) {
	a := newTestAPI(t)
	c := loginClient(t, a)
	resp, lines := openStream(t, c, a)
	for k, v := range map[string]string{
		"Content-Type":                "text/event-stream",
		"Cache-Control":               "no-cache",
		"X-Accel-Buffering":           "no",
		"Access-Control-Allow-Origin": "*",
	} {
		if got := resp.Header.Get(k); got != v {
			t.Errorf("%s = %q", k, got)
		}
	}
	if d := nextData(t, lines); !allZero(d) || !strings.Contains(d, "crane_scores") {
		t.Errorf("initial %s", d)
	}
	if s, b := call(t, c, a, "POST", "/scores/increment/6"); s != 200 {
		t.Fatal(s, b)
	}
	if d := nextData(t, lines); !strings.Contains(d, `{"id":6,"points":1}`) {
		t.Errorf("after increment %s", d)
	}
	if s, b := call(t, c, a, "POST", "/reset"); s != 200 && s != 204 {
		t.Fatal(s, b)
	}
	if d := nextData(t, lines); !allZero(d) {
		t.Errorf("after reset %s", d)
	}
}

func TestStreamPing(t *testing.T) {
	a := newTestAPI(t)
	_, lines := openStream(t, loginClient(t, a), a)
	nextData(t, lines)
	timeout := time.After(2 * time.Second)
	for {
		select {
		case l, ok := <-lines:
			if !ok {
				t.Fatal("ended")
			}
			if l == ": ping" {
				return
			}
		case <-timeout:
			t.Fatal("no ping")
		}
	}
}

func waitEOF(t *testing.T, lines <-chan string, d time.Duration) {
	t.Helper()
	timeout := time.After(d)
	for {
		select {
		case _, ok := <-lines:
			if !ok {
				return
			}
		case <-timeout:
			t.Fatal("stream did not end")
		}
	}
}

func TestStreamClosedOnLogout(t *testing.T) {
	a := newTestAPI(t)
	c := loginClient(t, a)
	_, lines := openStream(t, c, a)
	nextData(t, lines)
	if s, b := call(t, c, a, "POST", "/admin/logout"); s != 204 && s != 200 {
		t.Fatal(s, b)
	}
	waitEOF(t, lines, time.Second)
}

func TestStreamClosedOnExpiry(t *testing.T) {
	a := newTestAPI(t)
	T := time.Now().Add(1100 * time.Millisecond).Truncate(time.Second)
	a.Clock.Set(T.Add(-auth.TTL))
	tok, exp, err := a.Auth.Create()
	if err != nil || !exp.Equal(T) {
		t.Fatal(err, exp, T)
	}
	a.Clock.Set(time.Now())
	req, _ := http.NewRequest("GET", a.URL+"/scores/stream", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal(resp.StatusCode)
	}
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
	}
	now := time.Now()
	if now.Before(T.Add(-50 * time.Millisecond)) {
		t.Errorf("ended %v early", T.Sub(now))
	}
	if now.After(T.Add(2 * time.Second)) {
		t.Errorf("ended %v late", now.Sub(T))
	}
}

type stuckWriter struct {
	mu       sync.Mutex
	deadline time.Time
	h        http.Header
}

func (w *stuckWriter) Header() http.Header { return w.h }
func (w *stuckWriter) WriteHeader(int)     {}
func (w *stuckWriter) Flush()              {}
func (w *stuckWriter) SetWriteDeadline(t time.Time) error {
	w.mu.Lock()
	w.deadline = t
	w.mu.Unlock()
	return nil
}
func (w *stuckWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	d := w.deadline
	w.mu.Unlock()
	time.Sleep(time.Until(d))
	return 0, os.ErrDeadlineExceeded
}

func TestStuckClientDropped(t *testing.T) {
	ta := newTestAPI(t)
	tok, _, err := ta.Auth.Create()
	if err != nil {
		t.Fatal(err)
	}
	ta.Clock.Set(time.Now())
	api := &api{Deps: Deps{
		State: ta.State, Auth: ta.Auth, Broadcaster: ta.Broadcaster,
		StreamWriteTimeout: 200 * time.Millisecond, PingInterval: 50 * time.Millisecond,
		Now: ta.Clock.Now,
	}}
	req := httptest.NewRequest("GET", "/scores/stream", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
	done := make(chan struct{})
	start := time.Now()
	go func() {
		defer close(done)
		api.stream(&stuckWriter{h: http.Header{}}, req)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return")
	}
	if d := time.Since(start); d > 200*time.Millisecond+500*time.Millisecond {
		t.Errorf("took %v", d)
	}
	if n := ta.Broadcaster.Count(); n != 0 {
		t.Errorf("count %d", n)
	}
}

func TestStreamDoesNotBlockWrites(t *testing.T) {
	a := newTestAPI(t)
	c := loginClient(t, a)
	for i := 0; i < 20; i++ {
		resp, err := c.Get(a.URL + "/scores/stream")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { resp.Body.Close() })
	}
	start := time.Now()
	for i := 0; i < 200; i++ {
		if s, b := call(t, c, a, "POST", "/scores/increment/6"); s != 200 {
			t.Fatal(s, b)
		}
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("200 increments took %v", d)
	}
}
