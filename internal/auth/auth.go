// Package auth holds admin login sessions in memory, written through to a
// store so they survive a restart. Only SHA-256 hashes of tokens are kept.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"sync"
	"time"
)

const (
	CookieName = "d3_admin"
	TTL        = 7 * 24 * time.Hour
)

// SessionStore is the persistence the Manager needs (satisfied by *store.Store).
type SessionStore interface {
	Sessions() (map[[32]byte]time.Time, error)
	PutSession([32]byte, time.Time) error
	DeleteSession([32]byte) error
	DeleteAllSessions() error
	PasswordFingerprint() ([]byte, error)
	SetPasswordFingerprint([]byte) error
}

// Session is a snapshot of a live session. Done is closed when it is deleted.
type Session struct {
	Expires time.Time
	Done    <-chan struct{}
}

type entry struct {
	expires time.Time
	done    chan struct{}
}

type Manager struct {
	st     SessionStore
	now    func() time.Time
	pwHash [32]byte

	mu       sync.Mutex
	sessions map[[32]byte]*entry
}

const saltLen = 16

func fingerprint(salt []byte, password string) []byte {
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(password))
	return h.Sum(append([]byte(nil), salt...))
}

// New loads sessions from st. If the stored password fingerprint is missing or
// does not match password, every session is deleted and a new fingerprint is
// stored. Expired sessions are deleted.
func New(st SessionStore, password string, now func() time.Time) (*Manager, error) {
	m := &Manager{st: st, now: now, pwHash: sha256.Sum256([]byte(password)), sessions: map[[32]byte]*entry{}}

	fp, err := st.PasswordFingerprint()
	if err != nil {
		return nil, err
	}
	match := len(fp) == saltLen+sha256.Size &&
		subtle.ConstantTimeCompare(fp, fingerprint(fp[:saltLen], password)) == 1
	if !match {
		if err := st.DeleteAllSessions(); err != nil {
			return nil, err
		}
		salt := make([]byte, saltLen)
		if _, err := rand.Read(salt); err != nil {
			return nil, err
		}
		if err := st.SetPasswordFingerprint(fingerprint(salt, password)); err != nil {
			return nil, err
		}
	}

	stored, err := st.Sessions()
	if err != nil {
		return nil, err
	}
	t := now()
	for h, exp := range stored {
		if !t.Before(exp) {
			if err := st.DeleteSession(h); err != nil {
				return nil, err
			}
			continue
		}
		m.sessions[h] = &entry{expires: exp, done: make(chan struct{})}
	}
	return m, nil
}

// CheckPassword compares in constant time.
func (m *Manager) CheckPassword(pw string) bool {
	h := sha256.Sum256([]byte(pw))
	return subtle.ConstantTimeCompare(h[:], m.pwHash[:]) == 1
}

// Create makes a session valid for TTL. Expiry is truncated to the second,
// the store's precision, so it is identical after a restart.
func (m *Manager) Create() (token string, expires time.Time, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	expires = m.now().Add(TTL).Truncate(time.Second)
	h := sha256.Sum256(raw)

	m.mu.Lock()
	defer m.mu.Unlock()
	if err = m.st.PutSession(h, expires); err != nil {
		return "", time.Time{}, err
	}
	m.sessions[h] = &entry{expires: expires, done: make(chan struct{})}
	return token, expires, nil
}

func hashToken(token string) ([32]byte, bool) {
	var h [32]byte
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != 32 {
		return h, false
	}
	return sha256.Sum256(raw), true
}

// Lookup returns the session for token. An expired session is deleted lazily.
func (m *Manager) Lookup(token string) (Session, bool) {
	h, ok := hashToken(token)
	if !ok {
		return Session{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.sessions[h]
	if !ok {
		return Session{}, false
	}
	if !m.now().Before(e.expires) {
		delete(m.sessions, h)
		close(e.done)
		_ = m.st.DeleteSession(h)
		return Session{}, false
	}
	return Session{Expires: e.expires, Done: e.done}, true
}

// Delete removes the session and closes its Done. An unknown token is not an error.
func (m *Manager) Delete(token string) error {
	h, ok := hashToken(token)
	if !ok {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.sessions[h]
	if !ok {
		return nil
	}
	if err := m.st.DeleteSession(h); err != nil {
		return err
	}
	delete(m.sessions, h)
	close(e.done)
	return nil
}
