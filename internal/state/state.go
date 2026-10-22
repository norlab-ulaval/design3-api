// Package state keeps the in-memory copy of the competition data. Reads are
// lock-free (an atomic pointer to an immutable snapshot holding prebuilt JSON);
// writes are serialized, committed to SQLite first, then swapped in.
package state

import (
	"encoding/json"
	"errors"
	"slices"
	"sort"
	"sync"
	"sync/atomic"

	"design3api/internal/broadcast"
	"design3api/internal/store"
)

var (
	ErrUnknownTeam = errors.New("unknown team")
	ErrAlreadyZero = errors.New("team score is already 0")
)

type Crane struct {
	ID       int `json:"id"`
	NbTokens int `json:"nb_tokens"`
}

type Vehicle struct {
	ID   int      `json:"id"`
	Path []string `json:"path"`
}

type Score struct {
	ID     int `json:"id"`
	Points int `json:"points"`
}

// snapshot is immutable once stored in State.snap.
type snapshot struct {
	cranes   map[int]int
	vehicles map[int][]string
	scores   map[int]int // includes rows of teams no longer configured

	cranesJSON     []byte
	vehiclesJSON   []byte
	craneJSON      map[int][]byte
	vehicleJSON    map[int][]byte
	scoreboardJSON []byte
}

type State struct {
	st    *store.Store
	teams Teams
	bc    *broadcast.Broadcaster

	writeMu sync.Mutex
	snap    atomic.Pointer[snapshot]
}

// New makes sure every configured team has a score row, then loads the data.
func New(st *store.Store, teams Teams, bc *broadcast.Broadcaster) (*State, error) {
	if err := st.EnsureScores(teams.IDs()); err != nil {
		return nil, err
	}
	d, err := st.Load()
	if err != nil {
		return nil, err
	}
	s := &State{st: st, teams: teams, bc: bc}
	sn := &snapshot{
		cranes:      d.Cranes,
		vehicles:    d.Vehicles,
		scores:      d.Scores,
		craneJSON:   map[int][]byte{},
		vehicleJSON: map[int][]byte{},
	}
	for id, n := range d.Cranes {
		sn.craneJSON[id] = mustJSON(Crane{id, n})
	}
	for id, p := range d.Vehicles {
		sn.vehicleJSON[id] = mustJSON(Vehicle{id, nonNil(p)})
	}
	s.buildCranes(sn)
	s.buildVehicles(sn)
	s.buildScoreboard(sn)
	s.snap.Store(sn)
	return s, nil
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err) // plain structs of ints and strings cannot fail
	}
	return b
}

func nonNil(p []string) []string {
	if p == nil {
		return []string{}
	}
	return p
}

func sortedKeys[V any](m map[int]V) []int {
	ids := make([]int, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// join builds {"key":[item,item,...]} from prebuilt item bytes, in id order.
func join(key string, ids []int, items map[int][]byte) []byte {
	b := []byte(`{"` + key + `":[`)
	for i, id := range ids {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, items[id]...)
	}
	return append(b, ']', '}')
}

func (s *State) buildCranes(sn *snapshot) {
	sn.cranesJSON = join("cranes", sortedKeys(sn.cranes), sn.craneJSON)
}

func (s *State) buildVehicles(sn *snapshot) {
	sn.vehiclesJSON = join("vehicles", sortedKeys(sn.vehicles), sn.vehicleJSON)
}

func (s *State) buildScoreboard(sn *snapshot) {
	cs := make([]Score, 0, s.teams.Cranes)
	vs := make([]Score, 0, s.teams.Vehicles)
	for id := 1; id <= s.teams.Cranes+s.teams.Vehicles; id++ {
		sc := Score{id, sn.scores[id]}
		if id <= s.teams.Cranes {
			cs = append(cs, sc)
		} else {
			vs = append(vs, sc)
		}
	}
	sn.scoreboardJSON = mustJSON(struct {
		CraneScores   []Score `json:"crane_scores"`
		VehicleScores []Score `json:"vehicle_scores"`
	}{cs, vs})
}

// Reads. The returned bytes are shared: callers must not modify them.

func (s *State) CranesJSON() []byte     { return s.snap.Load().cranesJSON }
func (s *State) VehiclesJSON() []byte   { return s.snap.Load().vehiclesJSON }
func (s *State) ScoreboardJSON() []byte { return s.snap.Load().scoreboardJSON }

func (s *State) CraneJSON(id int) ([]byte, bool) {
	b, ok := s.snap.Load().craneJSON[id]
	return b, ok
}

func (s *State) VehicleJSON(id int) ([]byte, bool) {
	b, ok := s.snap.Load().vehicleJSON[id]
	return b, ok
}

// Overview returns every crane and vehicle sorted by id.
func (s *State) Overview() ([]Crane, []Vehicle) {
	sn := s.snap.Load()
	cs := make([]Crane, 0, len(sn.cranes))
	for _, id := range sortedKeys(sn.cranes) {
		cs = append(cs, Crane{id, sn.cranes[id]})
	}
	vs := make([]Vehicle, 0, len(sn.vehicles))
	for _, id := range sortedKeys(sn.vehicles) {
		vs = append(vs, Vehicle{id, slices.Clone(sn.vehicles[id])})
	}
	return cs, vs
}

// Writes.

func (s *State) SetCrane(id, nb int) ([]byte, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	old := s.snap.Load()
	if cur, ok := old.cranes[id]; ok && cur == nb {
		return old.craneJSON[id], nil
	}
	if err := s.st.PutCrane(id, nb); err != nil {
		return nil, err
	}
	sn := *old
	sn.cranes = cloneMap(old.cranes)
	sn.cranes[id] = nb
	sn.craneJSON = cloneMap(old.craneJSON)
	item := mustJSON(Crane{id, nb})
	sn.craneJSON[id] = item
	s.buildCranes(&sn)
	s.snap.Store(&sn)
	return item, nil
}

func (s *State) SetVehicle(id int, path []string) ([]byte, error) {
	path = slices.Clone(nonNil(path))
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	old := s.snap.Load()
	if cur, ok := old.vehicles[id]; ok && slices.Equal(cur, path) {
		return old.vehicleJSON[id], nil
	}
	if err := s.st.PutVehicle(id, path); err != nil {
		return nil, err
	}
	sn := *old
	sn.vehicles = cloneMap(old.vehicles)
	sn.vehicles[id] = path
	sn.vehicleJSON = cloneMap(old.vehicleJSON)
	item := mustJSON(Vehicle{id, path})
	sn.vehicleJSON[id] = item
	s.buildVehicles(&sn)
	s.snap.Store(&sn)
	return item, nil
}

// AddPoints adds delta to a team's score and returns the new scoreboard JSON.
func (s *State) AddPoints(teamID, delta int) ([]byte, error) {
	if !s.teams.Valid(teamID) {
		return nil, ErrUnknownTeam
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	old := s.snap.Load()
	if old.scores[teamID]+delta < 0 {
		return nil, ErrAlreadyZero
	}
	if err := s.st.AddPoints(teamID, delta); err != nil {
		return nil, err
	}
	sn := *old
	sn.scores = cloneMap(old.scores)
	sn.scores[teamID] += delta
	s.buildScoreboard(&sn)
	s.snap.Store(&sn)
	s.bc.Publish(sn.scoreboardJSON)
	return sn.scoreboardJSON, nil
}

// Reset deletes cranes and vehicles and zeroes every score.
func (s *State) Reset() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.st.Reset(); err != nil {
		return err
	}
	old := s.snap.Load()
	sn := &snapshot{
		cranes:      map[int]int{},
		vehicles:    map[int][]string{},
		scores:      make(map[int]int, len(old.scores)),
		craneJSON:   map[int][]byte{},
		vehicleJSON: map[int][]byte{},
	}
	for id := range old.scores {
		sn.scores[id] = 0
	}
	s.buildCranes(sn)
	s.buildVehicles(sn)
	s.buildScoreboard(sn)
	s.snap.Store(sn)
	s.bc.Publish(sn.scoreboardJSON)
	return nil
}

func cloneMap[V any](m map[int]V) map[int]V {
	c := make(map[int]V, len(m)+1)
	for k, v := range m {
		c[k] = v
	}
	return c
}
