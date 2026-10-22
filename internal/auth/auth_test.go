package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"path/filepath"
	"testing"
	"time"

	"design3api/internal/store"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func setup(t *testing.T, pw string) (*Manager, *store.Store, *clock) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	c := &clock{t: time.Unix(1_800_000_000, 0)}
	m, err := New(st, pw, c.now)
	if err != nil {
		t.Fatal(err)
	}
	return m, st, c
}

func TestCreateLookup(t *testing.T) {
	m, _, c := setup(t, "pw")
	tok, exp, err := m.Create()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil || len(raw) != 32 {
		t.Fatalf("bad token %q: %v len=%d", tok, err, len(raw))
	}
	s, ok := m.Lookup(tok)
	if !ok || !s.Expires.Equal(c.t.Add(TTL)) || !exp.Equal(s.Expires) {
		t.Fatalf("lookup = %v %v, exp %v", s, ok, exp)
	}
	if _, ok := m.Lookup("garbage"); ok {
		t.Fatal("garbage accepted")
	}
	if _, ok := m.Lookup(""); ok {
		t.Fatal("empty accepted")
	}
}

func TestOnlyHashStored(t *testing.T) {
	m, st, _ := setup(t, "pw")
	tok, _, _ := m.Create()
	raw, _ := base64.RawURLEncoding.DecodeString(tok)
	var rawKey [32]byte
	copy(rawKey[:], raw)
	got, err := st.Sessions()
	if err != nil || len(got) != 1 {
		t.Fatalf("sessions %v %v", got, err)
	}
	if _, ok := got[rawKey]; ok {
		t.Fatal("raw token stored")
	}
	if _, ok := got[sha256.Sum256(raw)]; !ok {
		t.Fatal("hash not stored")
	}
}

func TestExpiry(t *testing.T) {
	m, st, c := setup(t, "pw")
	tok, _, _ := m.Create()
	c.t = c.t.Add(TTL + time.Second)
	if _, ok := m.Lookup(tok); ok {
		t.Fatal("expired accepted")
	}
	if got, _ := st.Sessions(); len(got) != 0 {
		t.Fatalf("row remains: %v", got)
	}
}

func TestDeleteClosesDone(t *testing.T) {
	m, _, _ := setup(t, "pw")
	tok, _, _ := m.Create()
	s, _ := m.Lookup(tok)
	select {
	case <-s.Done:
		t.Fatal("done closed early")
	default:
	}
	if err := m.Delete(tok); err != nil {
		t.Fatal(err)
	}
	select {
	case <-s.Done:
	default:
		t.Fatal("done not closed")
	}
	if _, ok := m.Lookup(tok); ok {
		t.Fatal("still valid")
	}
	if err := m.Delete(tok); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete("garbage"); err != nil {
		t.Fatal(err)
	}
}

func TestSurvivesRestart(t *testing.T) {
	m, st, c := setup(t, "pw")
	tok, _, _ := m.Create()
	m2, err := New(st, "pw", c.now)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := m2.Lookup(tok); !ok || !s.Expires.Equal(c.t.Add(TTL)) {
		t.Fatal("session lost")
	}
}

func TestPasswordChangeLogsOut(t *testing.T) {
	m, st, c := setup(t, "pw")
	tok, _, _ := m.Create()
	m2, err := New(st, "other", c.now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m2.Lookup(tok); ok {
		t.Fatal("session survived")
	}
	if got, _ := st.Sessions(); len(got) != 0 {
		t.Fatal("rows remain")
	}
	tok2, _, _ := m2.Create()
	m3, _ := New(st, "other", c.now)
	if _, ok := m3.Lookup(tok2); !ok {
		t.Fatal("same password restart lost session")
	}
}

func TestExpiredPurgedAtStart(t *testing.T) {
	m, st, c := setup(t, "pw")
	m.Create()
	c.t = c.t.Add(TTL + time.Hour)
	if _, err := New(st, "pw", c.now); err != nil {
		t.Fatal(err)
	}
	if got, _ := st.Sessions(); len(got) != 0 {
		t.Fatal("expired row remains")
	}
}

func TestCheckPassword(t *testing.T) {
	m, _, _ := setup(t, "secret")
	if !m.CheckPassword("secret") {
		t.Fatal("right rejected")
	}
	for _, p := range []string{"wrong", "", "secre", "secret2"} {
		if m.CheckPassword(p) {
			t.Fatalf("%q accepted", p)
		}
	}
}
