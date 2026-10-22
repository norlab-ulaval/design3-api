// Package store persists cranes, vehicles, scores, admin sessions and a
// password fingerprint in SQLite.
package store

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "modernc.org/sqlite"
)

// Data is the full persisted game state. Scores holds every row, configured or not.
type Data struct {
	Cranes   map[int]int
	Vehicles map[int][]string
	Scores   map[int]int
}

// Store is a SQLite-backed store.
type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS cranes(id INTEGER PRIMARY KEY, nb_tokens INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS vehicles(id INTEGER PRIMARY KEY, path TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS scores(team_id INTEGER PRIMARY KEY, points INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS sessions(token_hash BLOB PRIMARY KEY, expires_at INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS meta(key TEXT PRIMARY KEY, value BLOB NOT NULL);
`

const fingerprintKey = "password_fingerprint"

// Open opens (creating if needed) the database at path.
func Open(path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Load reads all cranes, vehicles and scores.
func (s *Store) Load() (Data, error) {
	d := Data{Cranes: map[int]int{}, Vehicles: map[int][]string{}, Scores: map[int]int{}}

	rows, err := s.db.Query("SELECT id, nb_tokens FROM cranes")
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var id, n int
		if err := rows.Scan(&id, &n); err != nil {
			rows.Close()
			return d, err
		}
		d.Cranes[id] = n
	}
	if err := closeRows(rows); err != nil {
		return d, err
	}

	rows, err = s.db.Query("SELECT id, path FROM vehicles")
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var id int
		var raw string
		if err := rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return d, err
		}
		path := []string{}
		if err := json.Unmarshal([]byte(raw), &path); err != nil {
			rows.Close()
			return d, err
		}
		if path == nil {
			path = []string{}
		}
		d.Vehicles[id] = path
	}
	if err := closeRows(rows); err != nil {
		return d, err
	}

	rows, err = s.db.Query("SELECT team_id, points FROM scores")
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var id, p int
		if err := rows.Scan(&id, &p); err != nil {
			rows.Close()
			return d, err
		}
		d.Scores[id] = p
	}
	return d, closeRows(rows)
}

func closeRows(rows *sql.Rows) error {
	err := rows.Err()
	if cerr := rows.Close(); err == nil {
		err = cerr
	}
	return err
}

// EnsureScores inserts a zero-point row for each team that has none.
func (s *Store) EnsureScores(teamIDs []int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, id := range teamIDs {
		if _, err := tx.Exec("INSERT OR IGNORE INTO scores(team_id, points) VALUES(?, 0)", id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) PutCrane(id, nbTokens int) error {
	_, err := s.db.Exec("INSERT INTO cranes(id, nb_tokens) VALUES(?, ?) ON CONFLICT(id) DO UPDATE SET nb_tokens = excluded.nb_tokens", id, nbTokens)
	return err
}

func (s *Store) PutVehicle(id int, path []string) error {
	if path == nil {
		path = []string{}
	}
	b, err := json.Marshal(path)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("INSERT INTO vehicles(id, path) VALUES(?, ?) ON CONFLICT(id) DO UPDATE SET path = excluded.path", id, string(b))
	return err
}

func (s *Store) AddPoints(teamID, delta int) error {
	_, err := s.db.Exec("UPDATE scores SET points = points + ? WHERE team_id = ?", delta, teamID)
	return err
}

// Reset clears cranes and vehicles and zeroes every score, atomically. Sessions are untouched.
func (s *Store) Reset() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{"DELETE FROM cranes", "DELETE FROM vehicles", "UPDATE scores SET points = 0"} {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Sessions() (map[[32]byte]time.Time, error) {
	rows, err := s.db.Query("SELECT token_hash, expires_at FROM sessions")
	if err != nil {
		return nil, err
	}
	out := map[[32]byte]time.Time{}
	for rows.Next() {
		var h []byte
		var exp int64
		if err := rows.Scan(&h, &exp); err != nil {
			rows.Close()
			return nil, err
		}
		if len(h) != 32 {
			continue
		}
		var k [32]byte
		copy(k[:], h)
		out[k] = time.Unix(exp, 0)
	}
	if err := closeRows(rows); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) PutSession(hash [32]byte, expires time.Time) error {
	_, err := s.db.Exec("INSERT INTO sessions(token_hash, expires_at) VALUES(?, ?) ON CONFLICT(token_hash) DO UPDATE SET expires_at = excluded.expires_at", hash[:], expires.Unix())
	return err
}

func (s *Store) DeleteSession(hash [32]byte) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token_hash = ?", hash[:])
	return err
}

func (s *Store) DeleteAllSessions() error {
	_, err := s.db.Exec("DELETE FROM sessions")
	return err
}

// PasswordFingerprint returns nil, nil when none is stored.
func (s *Store) PasswordFingerprint() ([]byte, error) {
	var v []byte
	err := s.db.QueryRow("SELECT value FROM meta WHERE key = ?", fingerprintKey).Scan(&v)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Store) SetPasswordFingerprint(fp []byte) error {
	_, err := s.db.Exec("INSERT INTO meta(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", fingerprintKey, fp)
	return err
}
