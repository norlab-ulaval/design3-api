// Command api is the Design 3 scoreboard API.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"design3api/internal/auth"
	"design3api/internal/broadcast"
	"design3api/internal/config"
	"design3api/internal/httpapi"
	"design3api/internal/state"
	"design3api/internal/store"
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stderr))
}

func run(args []string, getenv func(string) string, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "healthcheck" {
		return healthcheck()
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		return 1
	}
	log := slog.New(slog.NewTextHandler(stderr, nil))

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		return 1
	}
	defer st.Close()

	teams := state.Teams{Cranes: cfg.CraneTeams, Vehicles: cfg.VehicleTeams}
	bc := broadcast.New()
	s, err := state.New(st, teams, bc)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		return 1
	}
	am, err := auth.New(st, cfg.Password, time.Now)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		return 1
	}
	handler := httpapi.New(httpapi.Deps{
		State: s, Auth: am, Broadcaster: bc, Teams: teams,
		LoginFailDelay: time.Second, StreamWriteTimeout: 10 * time.Second,
		PingInterval: 15 * time.Second, Now: time.Now, Log: log,
	})

	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := &http.Server{
		Addr: ":5000", Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second,
		BaseContext: func(net.Listener) context.Context { return base },
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("listening", "addr", srv.Addr, "crane_teams", cfg.CraneTeams,
		"vehicle_teams", cfg.VehicleTeams, "db_path", cfg.DBPath)

	select {
	case err := <-errc:
		fmt.Fprintf(stderr, "error: %s\n", err)
		return 1
	case <-sig:
	}
	cancel() // ends open streams
	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		log.Error("shutdown", "err", err)
	}
	log.Info("shutdown")
	return 0
}

func healthcheck() int {
	c := http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get("http://127.0.0.1:5000/cranes")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
