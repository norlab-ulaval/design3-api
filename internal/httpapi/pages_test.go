package httpapi

import (
	"strings"
	"testing"
)

func TestOverviewLists(t *testing.T) {
	a := newTestAPI(t)
	a.do(t, "POST", "/cranes/3", "application/json", `{"nb_tokens":0}`)
	a.do(t, "POST", "/vehicles/7", "application/json", `{"path":[]}`)
	code, h, body := a.do(t, "GET", "/", "", "")
	if code != 200 || h.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("got %d %q", code, h.Get("Content-Type"))
	}
	for _, want := range []string{"ZC3", "ZD7", "(aucun chemin)"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(body, "ZC4") {
		t.Error("ZC4 should be absent")
	}
	a.do(t, "POST", "/vehicles/7", "application/json", `{"path":["ZD7","<b>"]}`)
	_, _, body = a.do(t, "GET", "/", "", "")
	if strings.Contains(body, "<b>") || !strings.Contains(body, "&lt;b&gt;") {
		t.Error("path not escaped")
	}
	if !strings.Contains(body, "ZD7 → &lt;b&gt;") {
		t.Error("path not joined with arrow")
	}
}

func TestUnknownPathStillJSON404(t *testing.T) {
	a := newTestAPI(t)
	code, h, _ := a.do(t, "GET", "/nope", "", "")
	if code != 404 || h.Get("Content-Type") != "application/json" {
		t.Fatalf("got %d %q", code, h.Get("Content-Type"))
	}
}

func TestPagesSelfContained(t *testing.T) {
	a := newTestAPI(t)
	for _, p := range []string{"/", "/input_scores"} {
		_, _, body := a.do(t, "GET", p, "", "")
		if len(body) >= 30000 {
			t.Errorf("%s: %d bytes", p, len(body))
		}
		for _, bad := range []string{"http://", "https://", "//cdn", "localStorage"} {
			if strings.Contains(body, bad) {
				t.Errorf("%s contains %q", p, bad)
			}
		}
	}
}

func TestInputScoresServed(t *testing.T) {
	a := newTestAPI(t)
	code, h, body := a.do(t, "GET", "/input_scores", "", "")
	if code != 200 || !strings.HasPrefix(h.Get("Content-Type"), "text/html") {
		t.Fatalf("got %d %q", code, h.Get("Content-Type"))
	}
	for _, want := range []string{"Saisie des points", "touch-action: manipulation"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestPagesHiddenGuard(t *testing.T) {
	a := newTestAPI(t)
	for _, p := range []string{"/", "/input_scores"} {
		_, _, body := a.do(t, "GET", p, "", "")
		if !strings.Contains(body, "[hidden] { display: none !important; }") {
			t.Errorf("%s: no [hidden] guard", p)
		}
	}
}
