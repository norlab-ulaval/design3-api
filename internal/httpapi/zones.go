package httpapi

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
)

const maxTokens = 10000

var (
	digits = regexp.MustCompile(`^[0-9]+$`)
)

// digitID answers 404 for a non-digit {id}, whatever the method or headers.
func digitID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !digits.MatchString(r.PathValue("id")) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		next(w, r)
	}
}

// zoneID parses the {id} path value. ok is false when a response was written.
func zoneID(w http.ResponseWriter, r *http.Request, msg string) (id int, ok bool) {
	n, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || n < 1 || n > 99 {
		writeError(w, http.StatusBadRequest, msg)
		return 0, false
	}
	return n, true
}

func (a *api) cranesList() http.HandlerFunc {
	return methods(map[string]http.HandlerFunc{
		http.MethodGet: func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, a.State.CranesJSON())
		},
	})
}

func (a *api) vehiclesList() http.HandlerFunc {
	return methods(map[string]http.HandlerFunc{
		http.MethodGet: func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, a.State.VehiclesJSON())
		},
	})
}

func (a *api) crane() http.HandlerFunc {
	const msg = "loading_zone_id is invalid"
	return digitID(methods(map[string]http.HandlerFunc{
		http.MethodGet: func(w http.ResponseWriter, r *http.Request) {
			id, ok := zoneID(w, r, msg)
			if !ok {
				return
			}
			b, found := a.State.CraneJSON(id)
			if !found {
				writeError(w, http.StatusNotFound, "crane not found")
				return
			}
			writeJSON(w, http.StatusOK, b)
		},
		http.MethodPost: func(w http.ResponseWriter, r *http.Request) {
			if !isJSON(r) {
				writeError(w, http.StatusBadRequest, "Content type is not json")
				return
			}
			id, ok := zoneID(w, r, msg)
			if !ok {
				return
			}
			body, ok := readBody(w, r)
			if !ok {
				return
			}
			nb, valid := parseTokens(body)
			if !valid {
				writeError(w, http.StatusBadRequest, "nb_tokens is invalid")
				return
			}
			out, err := a.State.SetCrane(id, nb)
			if err != nil {
				a.internalError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, out)
		},
	}))
}

func (a *api) vehicle() http.HandlerFunc {
	const msg = "unloading_zone_id is invalid"
	return digitID(methods(map[string]http.HandlerFunc{
		http.MethodGet: func(w http.ResponseWriter, r *http.Request) {
			id, ok := zoneID(w, r, msg)
			if !ok {
				return
			}
			b, found := a.State.VehicleJSON(id)
			if !found {
				writeError(w, http.StatusNotFound, "vehicle not found")
				return
			}
			writeJSON(w, http.StatusOK, b)
		},
		http.MethodPost: func(w http.ResponseWriter, r *http.Request) {
			if !isJSON(r) {
				writeError(w, http.StatusBadRequest, "Content type is not json")
				return
			}
			id, ok := zoneID(w, r, msg)
			if !ok {
				return
			}
			body, ok := readBody(w, r)
			if !ok {
				return
			}
			path, valid := parsePath(body)
			if !valid {
				writeError(w, http.StatusBadRequest, "path must be a list of string")
				return
			}
			out, err := a.State.SetVehicle(id, path)
			if err != nil {
				a.internalError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, out)
		},
	}))
}

// field extracts the raw value of key from a JSON object body.
func field(body []byte, key string) (json.RawMessage, bool) {
	if !json.Valid(body) {
		return nil, false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, false
	}
	raw, ok := obj[key]
	return raw, ok
}

func parseTokens(body []byte) (int, bool) {
	raw, ok := field(body, "nb_tokens")
	if !ok || !digits.Match(raw) {
		return 0, false
	}
	n, err := strconv.Atoi(string(raw))
	if err != nil || n > maxTokens {
		return 0, false
	}
	return n, true
}

func parsePath(body []byte) ([]string, bool) {
	raw, ok := field(body, "path")
	if !ok || len(raw) == 0 || raw[0] != '[' {
		return nil, false
	}
	var items []*string
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false
	}
	out := make([]string, len(items))
	for i, p := range items {
		if p == nil {
			return nil, false
		}
		out[i] = *p
	}
	return out, true
}
