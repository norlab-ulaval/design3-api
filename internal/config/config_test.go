package config

import (
	"errors"
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadDefaults(t *testing.T) {
	got, err := Load(env(map[string]string{"SCOREBOARD_PASSWORD": "pw"}))
	if err != nil {
		t.Fatal(err)
	}
	want := Config{CraneTeams: 5, VehicleTeams: 8, Password: "pw", DBPath: "/data/design3.db"}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadOverrides(t *testing.T) {
	got, err := Load(env(map[string]string{
		"SCOREBOARD_PASSWORD": "pw", "CRANE_TEAMS": "3", "VEHICLE_TEAMS": "10", "DB_PATH": "/tmp/x.db",
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := Config{CraneTeams: 3, VehicleTeams: 10, Password: "pw", DBPath: "/tmp/x.db"}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadNoPassword(t *testing.T) {
	for _, m := range []map[string]string{{}, {"SCOREBOARD_PASSWORD": ""}} {
		if _, err := Load(env(m)); !errors.Is(err, ErrNoPassword) {
			t.Fatalf("env %v: got %v, want ErrNoPassword", m, err)
		}
	}
}

func TestLoadBadTeams(t *testing.T) {
	for _, name := range []string{"CRANE_TEAMS", "VEHICLE_TEAMS"} {
		for _, v := range []string{"abc", "-1", "100"} {
			_, err := Load(env(map[string]string{"SCOREBOARD_PASSWORD": "pw", name: v}))
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("%s=%q: got %v, want error naming %s", name, v, err, name)
			}
		}
		got, err := Load(env(map[string]string{"SCOREBOARD_PASSWORD": "pw", name: "0"}))
		if err != nil {
			t.Fatalf("%s=0: %v", name, err)
		}
		if got.CraneTeams != 0 && name == "CRANE_TEAMS" || got.VehicleTeams != 0 && name == "VEHICLE_TEAMS" {
			t.Fatalf("%s=0 not applied: %+v", name, got)
		}
	}
}
