package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"design3api/internal/auth"
)

const sessionMaxAge = 7 * 24 * 3600

// session returns the live session named by the d3_admin cookie.
func (a *api) session(r *http.Request) (auth.Session, bool) {
	c, err := r.Cookie(auth.CookieName)
	if err != nil {
		return auth.Session{}, false
	}
	return a.Auth.Lookup(c.Value)
}

func isHTTPS(r *http.Request) bool {
	return r.Header.Get("X-Forwarded-Proto") == "https"
}

// sameOrigin is true when there is no Origin header, or it names this host.
func (a *api) sameOrigin(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return true
	}
	scheme := "http"
	if isHTTPS(r) {
		scheme = "https"
	}
	return o == scheme+"://"+r.Host
}

// post wraps a POST handler with the origin check.
func (a *api) post(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.sameOrigin(r) {
			writeError(w, http.StatusForbidden, "Forbidden")
			return
		}
		h(w, r)
	}
}

// authed wraps a handler with the session check.
func (a *api) authed(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := a.session(r); !ok {
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		h(w, r)
	}
}

func (a *api) login() http.HandlerFunc {
	return methods(map[string]http.HandlerFunc{
		http.MethodPost: a.post(func(w http.ResponseWriter, r *http.Request) {
			if !isJSON(r) {
				writeError(w, http.StatusBadRequest, "Content type is not json")
				return
			}
			body, ok := readBody(w, r)
			if !ok {
				return
			}
			var req struct {
				Password string `json:"password"`
			}
			if json.Unmarshal(body, &req) != nil || !a.Auth.CheckPassword(req.Password) {
				select {
				case <-time.After(a.LoginFailDelay):
				case <-r.Context().Done():
					return
				}
				writeError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}
			token, expires, err := a.Auth.Create()
			if err != nil {
				a.internalError(w, err)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name: auth.CookieName, Value: token, Path: "/",
				Expires: expires, MaxAge: sessionMaxAge,
				HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: isHTTPS(r),
			})
			w.WriteHeader(http.StatusNoContent)
		}),
	})
}

func (a *api) logout() http.HandlerFunc {
	return methods(map[string]http.HandlerFunc{
		http.MethodPost: a.post(func(w http.ResponseWriter, r *http.Request) {
			if c, err := r.Cookie(auth.CookieName); err == nil {
				if err := a.Auth.Delete(c.Value); err != nil {
					a.internalError(w, err)
					return
				}
			}
			http.SetCookie(w, &http.Cookie{
				Name: auth.CookieName, Value: "", Path: "/", MaxAge: -1,
				HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: isHTTPS(r),
			})
			w.WriteHeader(http.StatusNoContent)
		}),
	})
}

func (a *api) sessionEndpoint() http.HandlerFunc {
	return methods(map[string]http.HandlerFunc{
		http.MethodGet: a.authed(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, []byte(`{"authenticated":true}`))
		}),
	})
}
