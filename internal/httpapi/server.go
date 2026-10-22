// Package httpapi is the net/http layer: routing, middleware and handlers.
package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"design3api/internal/auth"
	"design3api/internal/broadcast"
	"design3api/internal/state"
)

const maxBody = 1 << 16

// Deps are the collaborators and tunables of the API.
type Deps struct {
	State                                            *state.State
	Auth                                             *auth.Manager
	Broadcaster                                      *broadcast.Broadcaster
	Teams                                            state.Teams
	LoginFailDelay, StreamWriteTimeout, PingInterval time.Duration
	Now                                              func() time.Time
	Log                                              *slog.Logger
}

type api struct {
	Deps
}

// New builds the HTTP handler.
func New(d Deps) http.Handler {
	if d.Log == nil {
		d.Log = slog.Default()
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	a := &api{Deps: d}
	mux := http.NewServeMux()
	mux.HandleFunc("/cranes", a.cranesList())
	mux.HandleFunc("/cranes/{$}", a.cranesList())
	mux.HandleFunc("/cranes/{id}", a.crane())
	mux.HandleFunc("/vehicles", a.vehiclesList())
	mux.HandleFunc("/vehicles/{$}", a.vehiclesList())
	mux.HandleFunc("/vehicles/{id}", a.vehicle())
	mux.HandleFunc("/admin/login", a.login())
	mux.HandleFunc("/admin/logout", a.logout())
	mux.HandleFunc("/admin/session", a.sessionEndpoint())
	mux.HandleFunc("/scores", a.scoreboard())
	mux.HandleFunc("/scores/{$}", a.scoreboard())
	mux.HandleFunc("/scores/increment/{team}", a.scoreChange(1))
	mux.HandleFunc("/scores/decrement/{team}", a.scoreChange(-1))
	mux.HandleFunc("/scores/stream", methods(map[string]http.HandlerFunc{http.MethodGet: a.stream}))
	mux.HandleFunc("/reset", a.reset())
	mux.HandleFunc("/{$}", a.overview(mustParseIndex()))
	mux.HandleFunc("/input_scores", a.inputScores())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	return a.recoverMW(cors(limitBody(mux)))
}

func (a *api) recoverMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				a.Log.Error("panic", "panic", v, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			ah := r.Header.Get("Access-Control-Request-Headers")
			if ah == "" {
				ah = "Content-Type"
			}
			h.Set("Access-Control-Allow-Headers", ah)
			h.Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	b, _ := json.Marshal(map[string]string{"error": msg})
	writeJSON(w, status, b)
}

func isJSON(r *http.Request) bool {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mt == "application/json"
}

func methods(m map[string]http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h, ok := m[r.Method]; ok {
			h(w, r)
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *api) internalError(w http.ResponseWriter, err error) {
	a.Log.Error("internal error", "err", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

// readBody reads the whole (size-capped) request body, answering 413 itself
// when the cap is exceeded. ok is false when a response has been written.
func readBody(w http.ResponseWriter, r *http.Request) (b []byte, ok bool) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return nil, false
	}
	return b, true
}
