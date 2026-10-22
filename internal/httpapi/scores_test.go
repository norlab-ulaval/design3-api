package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type board struct {
	Crane   []struct{ ID, Points int } `json:"crane_scores"`
	Vehicle []struct{ ID, Points int } `json:"vehicle_scores"`
}

func call(t *testing.T, c *http.Client, a *testAPI, method, path string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(method, a.URL+path, nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(b))
}

func points(t *testing.T, c *http.Client, a *testAPI, id int) int {
	t.Helper()
	s, b := call(t, c, a, "GET", "/scores")
	if s != 200 {
		t.Fatal(s, b)
	}
	var bd board
	if err := json.Unmarshal([]byte(b), &bd); err != nil {
		t.Fatal(err)
	}
	for _, l := range [][]struct{ ID, Points int }{bd.Crane, bd.Vehicle} {
		for _, x := range l {
			if x.ID == id {
				return x.Points
			}
		}
	}
	t.Fatalf("team %d not in %s", id, b)
	return 0
}

func TestScoreboard(t *testing.T) {
	a := newTestAPI(t)
	c := loginClient(t, a)
	s, b := call(t, c, a, "GET", "/scores")
	want := `{"crane_scores":[{"id":1,"points":0},{"id":2,"points":0},{"id":3,"points":0},{"id":4,"points":0},{"id":5,"points":0}],` +
		`"vehicle_scores":[{"id":6,"points":0},{"id":7,"points":0},{"id":8,"points":0},{"id":9,"points":0},{"id":10,"points":0},{"id":11,"points":0},{"id":12,"points":0},{"id":13,"points":0}]}`
	if s != 200 || b != want {
		t.Errorf("%d %s", s, b)
	}
}

func TestIncrement(t *testing.T) {
	a := newTestAPI(t)
	c := loginClient(t, a)
	if s, b := call(t, c, a, "POST", "/scores/increment/6"); s != 200 || points(t, c, a, 6) != 1 {
		t.Fatal(s, b)
	}
	if s, b := call(t, c, a, "POST", "/scores/increment/6?amount=5"); s != 200 || points(t, c, a, 6) != 6 {
		t.Fatal(s, b)
	}
	for _, v := range []string{"0", "101", "abc", "-1", "2.5", ""} {
		s, b := call(t, c, a, "POST", "/scores/increment/6?amount="+v)
		if s != 400 || b != errBody("amount is invalid") {
			t.Errorf("amount=%q: %d %s", v, s, b)
		}
	}
	if points(t, c, a, 6) != 6 {
		t.Error("score changed by invalid amount")
	}
	if s, b := call(t, c, a, "POST", "/scores/increment/6?amount=100"); s != 200 || points(t, c, a, 6) != 106 {
		t.Fatal(s, b)
	}
}

func TestDecrement(t *testing.T) {
	a := newTestAPI(t)
	c := loginClient(t, a)
	s, b := call(t, c, a, "POST", "/scores/decrement/2")
	if s != 400 || b != errBody("team score is already 0") {
		t.Errorf("%d %s", s, b)
	}
	call(t, c, a, "POST", "/scores/increment/2")
	if s, b := call(t, c, a, "POST", "/scores/decrement/2"); s != 200 || points(t, c, a, 2) != 0 {
		t.Error(s, b)
	}
}

func TestDecrementIgnoresAmount(t *testing.T) {
	a := newTestAPI(t)
	c := loginClient(t, a)
	call(t, c, a, "POST", "/scores/increment/6?amount=3")
	if s, b := call(t, c, a, "POST", "/scores/decrement/6?amount=5"); s != 200 || points(t, c, a, 6) != 2 {
		t.Fatal(s, b)
	}
}

func TestScoreTeamIDs(t *testing.T) {
	a := newTestAPI(t)
	c := loginClient(t, a)
	for _, id := range []string{"14", "0", "99999999999999999999"} {
		for _, d := range []string{"increment", "decrement"} {
			s, b := call(t, c, a, "POST", "/scores/"+d+"/"+id)
			if s != 404 || b != errBody("team_id is invalid") {
				t.Errorf("%s %s: %d %s", d, id, s, b)
			}
		}
	}
	if s, _ := call(t, c, a, "POST", "/scores/increment/abc"); s != 404 {
		t.Error(s)
	}
	// no session: 401 even for a non-digit id (decided order)
	expect(t, a, "POST", "/scores/increment/abc", "", "", 401, errBody("Unauthorized"))
}

func TestReset(t *testing.T) {
	a := newTestAPI(t)
	c := loginClient(t, a)
	a.do(t, "POST", "/cranes/1", js, `{"nb_tokens":3}`)
	a.do(t, "POST", "/vehicles/1", js, `{"path":["A1"]}`)
	call(t, c, a, "POST", "/scores/increment/1")
	s, b := call(t, c, a, "POST", "/reset")
	if s != 200 || b != `{"message":"API reset was successful."}` {
		t.Fatal(s, b)
	}
	expect(t, a, "GET", "/cranes", "", "", 200, `{"cranes":[]}`)
	expect(t, a, "GET", "/vehicles", "", "", 200, `{"vehicles":[]}`)
	if points(t, c, a, 1) != 0 {
		t.Error("score not reset")
	}
	if s, _ := call(t, c, a, "GET", "/admin/session"); s != 200 {
		t.Errorf("session after reset: %d", s)
	}
}

func TestConcurrentClicksHTTP(t *testing.T) {
	a := newTestAPI(t)
	clients := []*http.Client{loginClient(t, a), loginClient(t, a)}
	var wg sync.WaitGroup
	for _, c := range clients {
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, _ := http.NewRequest("POST", a.URL+"/scores/increment/3", nil)
				resp, err := c.Do(req)
				if err != nil {
					t.Error(err)
					return
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode != 200 {
					t.Error(resp.StatusCode)
				}
			}()
		}
	}
	wg.Wait()
	if n := points(t, clients[0], a, 3); n != 100 {
		t.Errorf("got %d, want 100", n)
	}
}
