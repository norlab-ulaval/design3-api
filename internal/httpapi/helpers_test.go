package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"design3api/internal/auth"
	"design3api/internal/broadcast"
	"design3api/internal/state"
	"design3api/internal/store"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t
}

type testAPI struct {
	URL         string
	State       *state.State
	Auth        *auth.Manager
	Broadcaster *broadcast.Broadcaster
	Store       *store.Store
	Clock       *fakeClock
}

func newTestAPI(t *testing.T) *testAPI {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	clock := &fakeClock{t: time.Now()}
	teams := state.Teams{Cranes: 5, Vehicles: 8}
	bc := broadcast.New()
	s, err := state.New(st, teams, bc)
	if err != nil {
		t.Fatal(err)
	}
	am, err := auth.New(st, "secret-pw", clock.Now)
	if err != nil {
		t.Fatal(err)
	}
	h := New(Deps{
		State: s, Auth: am, Broadcaster: bc, Teams: teams,
		LoginFailDelay:     50 * time.Millisecond,
		StreamWriteTimeout: 200 * time.Millisecond,
		PingInterval:       50 * time.Millisecond,
		Now:                clock.Now,
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &testAPI{URL: srv.URL, State: s, Auth: am, Broadcaster: bc, Store: st, Clock: clock}
}

func (a *testAPI) do(t *testing.T, method, path, contentType, body string) (int, http.Header, string) {
	t.Helper()
	req, err := http.NewRequest(method, a.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := noRedirect.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, string(b)
}

func newReq(url, reqHeaders string) (*http.Response, error) {
	req, err := http.NewRequest("OPTIONS", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Access-Control-Request-Headers", reqHeaders)
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Origin", "http://example.com")
	resp, err := noRedirect.Do(req)
	if err == nil {
		resp.Body.Close()
	}
	return resp, err
}

var noRedirect = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}
