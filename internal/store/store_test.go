package store

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func openAt(t *testing.T, path string) *Store {
	t.Helper()
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestPersistence(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.db")
	s := openAt(t, p)
	must(t, s.EnsureScores([]int{1, 2, 6}))
	must(t, s.PutCrane(3, 7))
	must(t, s.PutVehicle(7, []string{"ZD7", "A1"}))
	must(t, s.PutVehicle(8, []string{}))
	must(t, s.AddPoints(6, 5))
	must(t, s.AddPoints(6, -1))
	must(t, s.Close())

	s = openAt(t, p)
	defer s.Close()
	got, err := s.Load()
	must(t, err)
	want := Data{
		Cranes:   map[int]int{3: 7},
		Vehicles: map[int][]string{7: {"ZD7", "A1"}, 8: {}},
		Scores:   map[int]int{1: 0, 2: 0, 6: 4},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	if got.Vehicles[8] == nil {
		t.Fatal("empty path must not be nil")
	}
}

func TestEnsureScoresKeepsPoints(t *testing.T) {
	s := openAt(t, filepath.Join(t.TempDir(), "d.db"))
	defer s.Close()
	must(t, s.EnsureScores([]int{1}))
	must(t, s.AddPoints(1, 3))
	must(t, s.EnsureScores([]int{1, 2}))
	d, err := s.Load()
	must(t, err)
	if d.Scores[1] != 3 || d.Scores[2] != 0 {
		t.Fatalf("scores %v", d.Scores)
	}
}

func TestReset(t *testing.T) {
	s := openAt(t, filepath.Join(t.TempDir(), "d.db"))
	defer s.Close()
	must(t, s.EnsureScores([]int{1, 2, 9}))
	must(t, s.AddPoints(1, 3))
	must(t, s.AddPoints(9, 4))
	must(t, s.EnsureScores([]int{1, 2}))
	must(t, s.PutCrane(1, 2))
	must(t, s.PutVehicle(1, []string{"A"}))
	exp := time.Unix(2000000000, 0)
	must(t, s.PutSession([32]byte{1}, exp))
	must(t, s.Reset())
	d, err := s.Load()
	must(t, err)
	if len(d.Cranes) != 0 || len(d.Vehicles) != 0 {
		t.Fatalf("not empty: %v", d)
	}
	if !reflect.DeepEqual(d.Scores, map[int]int{1: 0, 2: 0, 9: 0}) {
		t.Fatalf("scores %v", d.Scores)
	}
	ss, err := s.Sessions()
	must(t, err)
	if len(ss) != 1 {
		t.Fatalf("sessions %v", ss)
	}
}

func TestSessions(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.db")
	s := openAt(t, p)
	a, b := [32]byte{1}, [32]byte{2}
	exp := time.Unix(2000000000, 0)
	must(t, s.PutSession(a, exp))
	must(t, s.PutSession(b, exp.Add(time.Hour)))
	must(t, s.DeleteSession(a))
	must(t, s.Close())

	s = openAt(t, p)
	defer s.Close()
	got, err := s.Sessions()
	must(t, err)
	if len(got) != 1 || !got[b].Equal(exp.Add(time.Hour)) {
		t.Fatalf("got %v", got)
	}
	must(t, s.DeleteAllSessions())
	got, err = s.Sessions()
	must(t, err)
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestFingerprint(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.db")
	s := openAt(t, p)
	fp, err := s.PasswordFingerprint()
	if err != nil || fp != nil {
		t.Fatalf("got %v %v", fp, err)
	}
	must(t, s.SetPasswordFingerprint([]byte("abc")))
	must(t, s.SetPasswordFingerprint([]byte("abcd")))
	must(t, s.Close())
	s = openAt(t, p)
	defer s.Close()
	fp, err = s.PasswordFingerprint()
	must(t, err)
	if !bytes.Equal(fp, []byte("abcd")) {
		t.Fatalf("got %q", fp)
	}
}

func TestPragmas(t *testing.T) {
	s := openAt(t, filepath.Join(t.TempDir(), "d.db"))
	defer s.Close()
	var jm string
	var sy int
	must(t, s.db.QueryRow("PRAGMA journal_mode").Scan(&jm))
	must(t, s.db.QueryRow("PRAGMA synchronous").Scan(&sy))
	if jm != "wal" || sy != 1 {
		t.Fatalf("journal_mode=%q synchronous=%d", jm, sy)
	}
}
