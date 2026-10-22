package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"design3api/internal/state"
)

func (a *api) scoreboard() http.HandlerFunc {
	return methods(map[string]http.HandlerFunc{
		http.MethodGet: a.authed(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, a.State.ScoreboardJSON())
		}),
	})
}

// scoreChange handles increment (sign +1) and decrement (sign -1).
// Order: origin, session, team id, amount (increment only).
func (a *api) scoreChange(sign int) http.HandlerFunc {
	return methods(map[string]http.HandlerFunc{
		http.MethodPost: a.post(a.authed(func(w http.ResponseWriter, r *http.Request) {
			ts := r.PathValue("team")
			if !digits.MatchString(ts) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			team, err := strconv.Atoi(ts)
			if err != nil || !a.Teams.Valid(team) { // overflow counts as unknown
				writeError(w, http.StatusNotFound, "team_id is invalid")
				return
			}
			amount := 1
			// Only increment takes ?amount; decrement always removes exactly 1.
			if q := r.URL.Query(); sign > 0 && q.Has("amount") {
				s := q.Get("amount")
				n, err := strconv.Atoi(s)
				if !digits.MatchString(s) || err != nil || n < 1 || n > 100 {
					writeError(w, http.StatusBadRequest, "amount is invalid")
					return
				}
				amount = n
			}
			out, err := a.State.AddPoints(team, sign*amount)
			switch {
			case errors.Is(err, state.ErrUnknownTeam):
				writeError(w, http.StatusNotFound, "team_id is invalid")
			case errors.Is(err, state.ErrAlreadyZero):
				writeError(w, http.StatusBadRequest, "team score is already 0")
			case err != nil:
				a.internalError(w, err)
			default:
				writeJSON(w, http.StatusOK, out)
			}
		})),
	})
}

func (a *api) reset() http.HandlerFunc {
	return methods(map[string]http.HandlerFunc{
		http.MethodPost: a.post(a.authed(func(w http.ResponseWriter, r *http.Request) {
			if err := a.State.Reset(); err != nil {
				a.internalError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, []byte(`{"message":"API reset was successful."}`))
		})),
	})
}
