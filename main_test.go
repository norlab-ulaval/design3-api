package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunNoPassword(t *testing.T) {
	for _, set := range []bool{false, true} {
		db := filepath.Join(t.TempDir(), "x.db")
		env := map[string]string{"DB_PATH": db}
		if set {
			env["SCOREBOARD_PASSWORD"] = ""
		}
		var buf bytes.Buffer
		code := run(nil, func(k string) string { return env[k] }, &buf)
		want := "error: SCOREBOARD_PASSWORD is not set. Set it in .env (see .env.example).\n"
		if code != 1 || buf.String() != want {
			t.Fatalf("code=%d out=%q", code, buf.String())
		}
		if _, err := os.Stat(db); !os.IsNotExist(err) {
			t.Fatalf("db was created: %v", err)
		}
	}
}
