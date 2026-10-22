package httpapi

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
	"strings"

	"design3api/internal/state"
)

//go:embed pages/*
var pagesFS embed.FS

type vehicleRow struct {
	ID   int
	Path string
}

type overviewData struct {
	Cranes   []state.Crane
	Vehicles []vehicleRow
}

func mustParseIndex() *template.Template {
	return template.Must(template.ParseFS(pagesFS, "pages/index.html"))
}

func (a *api) overview(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		cs, vs := a.State.Overview()
		d := overviewData{Cranes: cs}
		for _, v := range vs {
			p := "(aucun chemin)"
			if len(v.Path) > 0 {
				p = strings.Join(v.Path, " → ")
			}
			d.Vehicles = append(d.Vehicles, vehicleRow{v.ID, p})
		}
		var buf bytes.Buffer
		if err := tpl.Execute(&buf, d); err != nil {
			a.internalError(w, err)
			return
		}
		writeHTML(w, buf.Bytes())
	}
}

func (a *api) inputScores() http.HandlerFunc {
	page, err := pagesFS.ReadFile("pages/input_scores.html")
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeHTML(w, page)
	}
}

func writeHTML(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(b)
}
