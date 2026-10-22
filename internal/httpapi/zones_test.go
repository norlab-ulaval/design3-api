package httpapi

import (
	"fmt"
	"strings"
	"testing"
)

const js = "application/json"

func errBody(msg string) string { return `{"error":"` + msg + `"}` }

func expect(t *testing.T, a *testAPI, method, path, ct, body string, status int, want string) {
	t.Helper()
	s, _, b := a.do(t, method, path, ct, body)
	if s != status || strings.TrimSpace(b) != want {
		t.Errorf("%s %s %q: got %d %q, want %d %q", method, path, body, s, b, status, want)
	}
}

func TestCranesContract(t *testing.T) {
	a := newTestAPI(t)
	expect(t, a, "GET", "/cranes", "", "", 200, `{"cranes":[]}`)
	expect(t, a, "GET", "/cranes/", "", "", 200, `{"cranes":[]}`)
	expect(t, a, "GET", "/cranes/3", "", "", 404, errBody("crane not found"))
	expect(t, a, "POST", "/cranes/3", js, `{"nb_tokens":0}`, 200, `{"id":3,"nb_tokens":0}`)
	expect(t, a, "GET", "/cranes/3", "", "", 200, `{"id":3,"nb_tokens":0}`)
	expect(t, a, "GET", "/cranes", "", "", 200, `{"cranes":[{"id":3,"nb_tokens":0}]}`)
}

func TestVehiclesContract(t *testing.T) {
	a := newTestAPI(t)
	expect(t, a, "GET", "/vehicles", "", "", 200, `{"vehicles":[]}`)
	expect(t, a, "GET", "/vehicles/", "", "", 200, `{"vehicles":[]}`)
	expect(t, a, "GET", "/vehicles/3", "", "", 404, errBody("vehicle not found"))
	expect(t, a, "POST", "/vehicles/3", js, `{"path":[]}`, 200, `{"id":3,"path":[]}`)
	expect(t, a, "GET", "/vehicles/3", "", "", 200, `{"id":3,"path":[]}`)
	expect(t, a, "GET", "/vehicles", "", "", 200, `{"vehicles":[{"id":3,"path":[]}]}`)
}

func TestZoneIDBounds(t *testing.T) {
	a := newTestAPI(t)
	for _, z := range []struct{ base, msg, body string }{
		{"cranes", "loading_zone_id is invalid", `{"nb_tokens":1}`},
		{"vehicles", "unloading_zone_id is invalid", `{"path":[]}`},
	} {
		for _, id := range []string{"0", "100", "99999999999999999999"} {
			expect(t, a, "GET", "/"+z.base+"/"+id, "", "", 400, errBody(z.msg))
			expect(t, a, "POST", "/"+z.base+"/"+id, js, z.body, 400, errBody(z.msg))
		}
		for _, id := range []string{"1", "99"} {
			s, _, b := a.do(t, "POST", "/"+z.base+"/"+id, js, z.body)
			if s != 200 {
				t.Errorf("POST %s/%s: %d %s", z.base, id, s, b)
			}
			s, _, b = a.do(t, "GET", "/"+z.base+"/"+id, "", "")
			if s != 200 {
				t.Errorf("GET %s/%s: %d %s", z.base, id, s, b)
			}
		}
		for _, id := range []string{"abc", "-1", "1.5", "3/"} {
			expect(t, a, "GET", "/"+z.base+"/"+id, "", "", 404, errBody("not found"))
			expect(t, a, "POST", "/"+z.base+"/"+id, js, z.body, 404, errBody("not found"))
			expect(t, a, "POST", "/"+z.base+"/"+id, "text/plain", z.body, 404, errBody("not found"))
			expect(t, a, "DELETE", "/"+z.base+"/"+id, "", "", 404, errBody("not found"))
			expect(t, a, "PUT", "/"+z.base+"/"+id, "", "", 404, errBody("not found"))
		}
	}
}

func TestCheckOrder(t *testing.T) {
	a := newTestAPI(t)
	expect(t, a, "POST", "/cranes/0", "text/plain", `{"nb_tokens":1}`, 400, errBody("Content type is not json"))
	expect(t, a, "POST", "/cranes/0", js, `garbage`, 400, errBody("loading_zone_id is invalid"))
}

func TestContentType(t *testing.T) {
	a := newTestAPI(t)
	expect(t, a, "POST", "/cranes/3", "application/json; charset=utf-8", `{"nb_tokens":3}`, 200, `{"id":3,"nb_tokens":3}`)
	for _, ct := range []string{"", "text/plain", "application/vnd.api+json"} {
		expect(t, a, "POST", "/cranes/4", ct, `{"nb_tokens":3}`, 400, errBody("Content type is not json"))
	}
}

func TestCraneBody(t *testing.T) {
	a := newTestAPI(t)
	for _, n := range []string{"0", "3", "10000"} {
		expect(t, a, "POST", "/cranes/3", js, `{"nb_tokens":`+n+`}`, 200, fmt.Sprintf(`{"id":3,"nb_tokens":%s}`, n))
	}
	expect(t, a, "POST", "/cranes/3", js, `{"nb_tokens": 3}`, 200, `{"id":3,"nb_tokens":3}`)
	b := newTestAPI(t)
	bad := []string{
		`{"nb_tokens":10001}`, `{"nb_tokens":-1}`, `{"nb_tokens":-0}`, `{"nb_tokens":3.7}`,
		`{"nb_tokens":3.0}`, `{"nb_tokens":3e0}`, `{"nb_tokens":1e2}`, `{"nb_tokens":"3"}`,
		`{"nb_tokens":true}`, `{"nb_tokens":null}`, `{"nb_tokens":[]}`, `{"nb_tokens":{}}`,
		`{}`, `{"x":1}`, `{"nb_tokens":3} x`, `[1]`, ``,
	}
	for _, body := range bad {
		expect(t, b, "POST", "/cranes/3", js, body, 400, errBody("nb_tokens is invalid"))
	}
	expect(t, b, "GET", "/cranes", "", "", 200, `{"cranes":[]}`)
}

func TestVehicleBody(t *testing.T) {
	a := newTestAPI(t)
	expect(t, a, "POST", "/vehicles/3", js, `{"path":[]}`, 200, `{"id":3,"path":[]}`)
	expect(t, a, "POST", "/vehicles/3", js, `{"path":["ZD7","A1","ZC1"]}`, 200, `{"id":3,"path":["ZD7","A1","ZC1"]}`)
	expect(t, a, "POST", "/vehicles/3", js, `{"path": [ "A1" ]}`, 200, `{"id":3,"path":["A1"]}`)
	b := newTestAPI(t)
	for _, body := range []string{
		`{"path":null}`, `{"path":"A1"}`, `{"path":["A1", null]}`, `{"path":["A1", 3]}`,
		`{"path":{}}`, `{}`, `{"x":[]}`, `{bad`, ``,
	} {
		expect(t, b, "POST", "/vehicles/3", js, body, 400, errBody("path must be a list of string"))
	}
	expect(t, b, "GET", "/vehicles", "", "", 200, `{"vehicles":[]}`)
}

func TestNoPhantomOnError(t *testing.T) {
	a := newTestAPI(t)
	expect(t, a, "POST", "/cranes/5", js, `{"nb_tokens":-1}`, 400, errBody("nb_tokens is invalid"))
	expect(t, a, "GET", "/cranes/5", "", "", 404, errBody("crane not found"))
	expect(t, a, "GET", "/cranes", "", "", 200, `{"cranes":[]}`)
}

func TestBodyLimit(t *testing.T) {
	a := newTestAPI(t)
	body := `{"nb_tokens":1,"pad":"` + strings.Repeat("x", 65537-len(`{"nb_tokens":1,"pad":""}`)) + `"}`
	if len(body) != 65537 {
		t.Fatal(len(body))
	}
	expect(t, a, "POST", "/cranes/3", js, body, 413, errBody("request body too large"))
}

func TestCORS(t *testing.T) {
	a := newTestAPI(t)
	a.do(t, "POST", "/cranes/3", js, `{"nb_tokens":1}`)
	for _, c := range [][3]string{
		{"GET", "/cranes", ""}, {"GET", "/cranes/3", ""}, {"GET", "/cranes/9", ""},
		{"POST", "/cranes/3", js}, {"POST", "/cranes/3", "text/plain"},
		{"GET", "/nope", ""}, {"DELETE", "/cranes/3", ""}, {"GET", "/vehicles", ""},
	} {
		_, h, _ := a.do(t, c[0], c[1], c[2], `{"nb_tokens":2}`)
		if h.Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("%v: missing ACAO", c)
		}
		if _, ok := h["Access-Control-Allow-Credentials"]; ok {
			t.Errorf("%v: credentials header set", c)
		}
	}
	for _, p := range []string{"/cranes/3", "/scores/increment/1"} {
		req := "content-type, authorization"
		r, err := newReq(a.URL+p, req)
		if err != nil {
			t.Fatal(err)
		}
		if r.StatusCode != 200 {
			t.Errorf("%s: %d", p, r.StatusCode)
		}
		h := r.Header
		if h.Get("Access-Control-Allow-Origin") != "*" ||
			h.Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" ||
			h.Get("Access-Control-Allow-Headers") != req {
			t.Errorf("%s: bad preflight headers %v", p, h)
		}
		if _, ok := h["Access-Control-Allow-Credentials"]; ok {
			t.Errorf("%s: credentials header set", p)
		}
	}
}

func TestJSONContentType(t *testing.T) {
	a := newTestAPI(t)
	for _, c := range [][3]string{
		{"GET", "/cranes", ""}, {"GET", "/cranes/3", ""}, {"POST", "/cranes/3", "text/plain"},
		{"POST", "/cranes/3", js}, {"GET", "/vehicles", ""}, {"GET", "/nope", ""},
		{"DELETE", "/cranes/3", ""}, {"GET", "/cranes/0", ""},
	} {
		_, h, _ := a.do(t, c[0], c[1], c[2], `{"nb_tokens":2}`)
		if h.Get("Content-Type") != "application/json" {
			t.Errorf("%v: Content-Type %q", c, h.Get("Content-Type"))
		}
	}
}

func TestMethodNotAllowed(t *testing.T) {
	a := newTestAPI(t)
	expect(t, a, "DELETE", "/cranes/3", "", "", 405, errBody("method not allowed"))
	expect(t, a, "PUT", "/vehicles", "", "", 405, errBody("method not allowed"))
}

func TestUnknownRoute(t *testing.T) {
	a := newTestAPI(t)
	expect(t, a, "GET", "/nope", "", "", 404, errBody("not found"))
}
