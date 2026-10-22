package state

// Teams is the configured team layout: cranes take ids 1..Cranes, vehicles
// the Vehicles ids after them.
type Teams struct {
	Cranes, Vehicles int
}

// IDs returns every configured team id, 1..Cranes+Vehicles.
func (t Teams) IDs() []int {
	ids := make([]int, t.Cranes+t.Vehicles)
	for i := range ids {
		ids[i] = i + 1
	}
	return ids
}

// Valid reports whether id is a configured team.
func (t Teams) Valid(id int) bool { return id >= 1 && id <= t.Cranes+t.Vehicles }
