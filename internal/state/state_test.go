package state

import (
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"design3api/internal/broadcast"
	"design3api/internal/store"
)

func open(t *testing.T, path string, teams Teams) (*State, *store.Store, *broadcast.Broadcaster) {
	t.Helper()
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	bc := broadcast.New()
	s, err := New(st, teams, bc)
	if err != nil {
		t.Fatal(err)
	}
	return s, st, bc
}

func fresh(t *testing.T) (*State, *store.Store, *broadcast.Broadcaster, string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "t.db")
	s, st, bc := open(t, p, Teams{5, 8})
	t.Cleanup(func() { st.Close() })
	return s, st, bc, p
}

func TestJSONShapes(t *testing.T) {
	s, _, _, _ := fresh(t)
	if got := string(s.CranesJSON()); got != `{"cranes":[]}` {
		t.Fatal(got)
	}
	if got := string(s.VehiclesJSON()); got != `{"vehicles":[]}` {
		t.Fatal(got)
	}
	if _, ok := s.CraneJSON(3); ok {
		t.Fatal("crane 3 should not exist")
	}
	if _, ok := s.VehicleJSON(6); ok {
		t.Fatal("vehicle 6 should not exist")
	}
	want := `{"crane_scores":[{"id":1,"points":0},{"id":2,"points":0},{"id":3,"points":0},{"id":4,"points":0},{"id":5,"points":0}],` +
		`"vehicle_scores":[{"id":6,"points":0},{"id":7,"points":0},{"id":8,"points":0},{"id":9,"points":0},{"id":10,"points":0},{"id":11,"points":0},{"id":12,"points":0},{"id":13,"points":0}]}`
	if got := string(s.ScoreboardJSON()); got != want {
		t.Fatalf("got %s", got)
	}
}

func TestSortedLists(t *testing.T) {
	s, _, _, _ := fresh(t)
	if _, err := s.SetCrane(9, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetCrane(2, 0); err != nil {
		t.Fatal(err)
	}
	want := `{"cranes":[{"id":2,"nb_tokens":0},{"id":9,"nb_tokens":1}]}`
	if got := string(s.CranesJSON()); got != want {
		t.Fatal(got)
	}
	cs, vs := s.Overview()
	if len(cs) != 2 || cs[0].ID != 2 || cs[1].ID != 9 || len(vs) != 0 {
		t.Fatal(cs, vs)
	}
}

func TestEmptyPathEncodesAsArray(t *testing.T) {
	s, st, _, p := fresh(t)
	b, err := s.SetVehicle(7, nil)
	if err != nil || string(b) != `{"id":7,"path":[]}` {
		t.Fatal(string(b), err)
	}
	if got := string(s.VehiclesJSON()); got != `{"vehicles":[{"id":7,"path":[]}]}` {
		t.Fatal(got)
	}
	st.Close()
	s2, st2, _ := open(t, p, Teams{5, 8})
	defer st2.Close()
	b, ok := s2.VehicleJSON(7)
	if !ok || string(b) != `{"id":7,"path":[]}` {
		t.Fatal(string(b), ok)
	}
}

func TestUnchangedSkipsDB(t *testing.T) {
	s, st, _, _ := fresh(t)
	if _, err := s.SetCrane(3, 4); err != nil {
		t.Fatal(err)
	}
	st.Close()
	b, err := s.SetCrane(3, 4)
	if err != nil || string(b) != `{"id":3,"nb_tokens":4}` {
		t.Fatal(string(b), err)
	}
	if _, err := s.SetCrane(3, 5); err == nil {
		t.Fatal("changed value should hit the closed DB and fail")
	}
	if b, _ := s.CraneJSON(3); string(b) != `{"id":3,"nb_tokens":4}` {
		t.Fatal("failed write must not change state:", string(b))
	}
}

func TestUnchangedVehicleSkipsDB(t *testing.T) {
	s, st, _, _ := fresh(t)
	if _, err := s.SetVehicle(6, []string{"A1", "B2"}); err != nil {
		t.Fatal(err)
	}
	st.Close()
	b, err := s.SetVehicle(6, []string{"A1", "B2"})
	if err != nil || string(b) != `{"id":6,"path":["A1","B2"]}` {
		t.Fatal(string(b), err)
	}
}

func TestDecrementAtZero(t *testing.T) {
	s, _, _, _ := fresh(t)
	if _, err := s.AddPoints(1, -1); !errors.Is(err, ErrAlreadyZero) {
		t.Fatal(err)
	}
	if points(t, s, 1) != 0 {
		t.Fatal("score changed")
	}
}

func TestRemovedTeamHidden(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.db")
	s, st, _ := open(t, p, Teams{5, 8})
	if _, err := s.AddPoints(13, 3); err != nil {
		t.Fatal(err)
	}
	st.Close()

	s, st, _ = open(t, p, Teams{5, 7})
	if bytesContains(s.ScoreboardJSON(), `"id":13`) {
		t.Fatal("team 13 should be hidden")
	}
	if _, err := s.AddPoints(13, 1); !errors.Is(err, ErrUnknownTeam) {
		t.Fatal(err)
	}
	st.Close()

	s, st, _ = open(t, p, Teams{5, 8})
	defer st.Close()
	if !bytesContains(s.ScoreboardJSON(), `{"id":13,"points":3}`) {
		t.Fatal(string(s.ScoreboardJSON()))
	}
}

func bytesContains(b []byte, sub string) bool {
	return len(sub) <= len(b) && (string(b) != "" && indexOf(string(b), sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func points(t *testing.T, s *State, id int) int {
	t.Helper()
	// Add 0 is not allowed through the API; read via a delta of +0 is not
	// needed: parse the snapshot directly.
	sn := s.snap.Load()
	return sn.scores[id]
}

func TestConcurrentScores(t *testing.T) {
	s, st, _, p := fresh(t)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); s.AddPoints(6, 1) }()
	}
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); s.AddPoints(7, 1) }()
		go func() { defer wg.Done(); s.AddPoints(7, 5) }()
	}
	wg.Wait()
	if _, err := s.AddPoints(8, 10); err != nil {
		t.Fatal(err)
	}
	var ok, zero int32
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.AddPoints(8, -1)
			switch {
			case err == nil:
				atomic.AddInt32(&ok, 1)
			case errors.Is(err, ErrAlreadyZero):
				atomic.AddInt32(&zero, 1)
			default:
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if ok != 10 || zero != 20 {
		t.Fatal(ok, zero)
	}
	if points(t, s, 6) != 100 || points(t, s, 7) != 300 || points(t, s, 8) != 0 {
		t.Fatal(points(t, s, 6), points(t, s, 7), points(t, s, 8))
	}
	st.Close()
	s2, st2, _ := open(t, p, Teams{5, 8})
	defer st2.Close()
	if points(t, s2, 6) != 100 || points(t, s2, 7) != 300 || points(t, s2, 8) != 0 {
		t.Fatal("disk mismatch")
	}
}

func take(sub *broadcast.Sub) []byte {
	select {
	case <-sub.Ready():
		return sub.Take()
	case <-time.After(2 * time.Second):
		return nil
	}
}

func TestPublishes(t *testing.T) {
	s, _, bc, _ := fresh(t)
	sub := bc.Subscribe()
	defer bc.Unsubscribe(sub)
	// Drain the initial value, if any.
	select {
	case <-sub.Ready():
		sub.Take()
	default:
	}
	if _, err := s.SetCrane(1, 2); err != nil {
		t.Fatal(err)
	}
	select {
	case <-sub.Ready():
		if sub.Take() != nil {
			t.Fatal("SetCrane must not publish")
		}
	case <-time.After(100 * time.Millisecond):
	}
	want, err := s.AddPoints(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := take(sub); string(got) != string(want) {
		t.Fatalf("got %s want %s", got, want)
	}
	if err := s.Reset(); err != nil {
		t.Fatal(err)
	}
	if got := take(sub); string(got) != string(s.ScoreboardJSON()) || got == nil {
		t.Fatalf("got %s", got)
	}
}

func TestReset(t *testing.T) {
	s, _, bc, _ := fresh(t)
	s.SetCrane(1, 2)
	s.SetVehicle(6, []string{"A"})
	s.AddPoints(6, 4)
	sub := bc.Subscribe()
	defer bc.Unsubscribe(sub)
	if err := s.Reset(); err != nil {
		t.Fatal(err)
	}
	if string(s.CranesJSON()) != `{"cranes":[]}` || string(s.VehiclesJSON()) != `{"vehicles":[]}` {
		t.Fatal("lists not empty")
	}
	if _, ok := s.CraneJSON(1); ok {
		t.Fatal("crane remains")
	}
	if points(t, s, 6) != 0 {
		t.Fatal("score remains")
	}
	if got := take(sub); string(got) != string(s.ScoreboardJSON()) {
		t.Fatalf("got %s", got)
	}
}

func TestTeams(t *testing.T) {
	tm := Teams{2, 3}
	ids := tm.IDs()
	if len(ids) != 5 || ids[0] != 1 || ids[4] != 5 {
		t.Fatal(ids)
	}
	if tm.Valid(0) || !tm.Valid(1) || !tm.Valid(5) || tm.Valid(6) {
		t.Fatal("Valid wrong")
	}
}
