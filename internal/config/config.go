// Package config reads the process configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"strconv"
)

// ErrNoPassword is returned by Load when SCOREBOARD_PASSWORD is unset or empty.
var ErrNoPassword = errors.New("SCOREBOARD_PASSWORD is not set. Set it in .env (see .env.example).")

// Config is the validated runtime configuration.
type Config struct {
	CraneTeams, VehicleTeams int
	Password, DBPath         string
}

// Load builds a Config from getenv (os.Getenv in production).
func Load(getenv func(string) string) (Config, error) {
	pw := getenv("SCOREBOARD_PASSWORD")
	if pw == "" {
		return Config{}, ErrNoPassword
	}
	crane, err := teams(getenv, "CRANE_TEAMS", 5)
	if err != nil {
		return Config{}, err
	}
	vehicle, err := teams(getenv, "VEHICLE_TEAMS", 8)
	if err != nil {
		return Config{}, err
	}
	db := getenv("DB_PATH")
	if db == "" {
		db = "/data/design3.db"
	}
	return Config{CraneTeams: crane, VehicleTeams: vehicle, Password: pw, DBPath: db}, nil
}

func teams(getenv func(string) string, name string, def int) (int, error) {
	s := getenv(name)
	if s == "" {
		return def, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 99 {
		return 0, fmt.Errorf("%s must be a whole number from 0 to 99", name)
	}
	return n, nil
}
