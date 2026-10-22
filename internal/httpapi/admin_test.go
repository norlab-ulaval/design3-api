package httpapi

import (
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
	"time"
)

const goodLogin = `{"password":"secret-pw"}`

func doH(t *testing.T, a *testAPI, method, path, ct, body string, hdr map[string]string) (int, http.Header, string) {
	t.Helper()
	req, _ := http.NewRequest(method, a.URL+path, strings.NewReader(body))
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := noRedirect.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return resp.StatusCode, resp.Header, sb.String()
}

// loginClient returns a client with a cookie jar, logged in.
func loginClient(t *testing.T, a *testAPI) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar, CheckRedirect: noRedirect.CheckRedirect}
	resp, err := c.Post(a.URL+"/admin/login", js, strings.NewReader(goodLogin))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		t.Fatalf("login: %d", resp.StatusCode)
	}
	return c
}

func cookieHeader(t *testing.T, a *testAPI) string {
	t.Helper()
	s, h, _ := a.do(t, "POST", "/admin/login", js, goodLogin)
	if s != 204 {
		t.Fatalf("login %d", s)
	}
	return strings.SplitN(h.Get("Set-Cookie"), ";", 2)[0]
}

func TestLoginOK(t *testing.T) {
	a := newTestAPI(t)
	s, h, _ := a.do(t, "POST", "/admin/login", js, goodLogin)
	if s != 204 {
		t.Fatal(s)
	}
	c := h.Get("Set-Cookie")
	for _, want := range []string{"d3_admin=", "HttpOnly", "SameSite=Strict", "Path=/", "Max-Age=604800"} {
		if !strings.Contains(c, want) {
			t.Errorf("cookie %q lacks %q", c, want)
		}
	}
	if strings.Contains(c, "Secure") {
		t.Errorf("unexpected Secure: %q", c)
	}
	_, h, _ = doH(t, a, "POST", "/admin/login", js, goodLogin, map[string]string{"X-Forwarded-Proto": "https"})
	if !strings.Contains(h.Get("Set-Cookie"), "Secure") {
		t.Errorf("want Secure: %q", h.Get("Set-Cookie"))
	}
}

func TestLoginWrong(t *testing.T) {
	a := newTestAPI(t)
	for _, body := range []string{`{"password":"nope"}`, `{}`, `{"password":5}`, `{"password":null}`} {
		start := time.Now()
		s, h, b := a.do(t, "POST", "/admin/login", js, body)
		if s != 401 || strings.TrimSpace(b) != errBody("Unauthorized") {
			t.Errorf("%s: %d %s", body, s, b)
		}
		if h.Get("Set-Cookie") != "" {
			t.Errorf("%s: cookie set", body)
		}
		if time.Since(start) < 50*time.Millisecond {
			t.Errorf("%s: answered too fast", body)
		}
	}
	done := make(chan struct{})
	go func() {
		a.do(t, "POST", "/admin/login", js, `{"password":"nope"}`)
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)
	start := time.Now()
	if s, _, _ := a.do(t, "GET", "/cranes", "", ""); s != 200 {
		t.Error(s)
	}
	if d := time.Since(start); d > 20*time.Millisecond {
		t.Errorf("GET /cranes took %v while login waited", d)
	}
	<-done
	start = time.Now()
	expect(t, a, "POST", "/admin/login", "text/plain", goodLogin, 400, errBody("Content type is not json"))
	if time.Since(start) >= 50*time.Millisecond {
		t.Error("non-json content type was delayed")
	}
}

func TestSessionEndpoint(t *testing.T) {
	a := newTestAPI(t)
	expect(t, a, "GET", "/admin/session", "", "", 401, errBody("Unauthorized"))
	ck := cookieHeader(t, a)
	s, _, b := doH(t, a, "GET", "/admin/session", "", "", map[string]string{"Cookie": ck})
	if s != 200 || strings.TrimSpace(b) != `{"authenticated":true}` {
		t.Errorf("%d %s", s, b)
	}
}

func TestLogout(t *testing.T) {
	a := newTestAPI(t)
	ck := cookieHeader(t, a)
	hdr := map[string]string{"Cookie": ck}
	s, h, _ := doH(t, a, "POST", "/admin/logout", "", "", hdr)
	if s != 204 || !strings.Contains(h.Get("Set-Cookie"), "Max-Age=0") {
		t.Errorf("%d %q", s, h.Get("Set-Cookie"))
	}
	for _, p := range []string{"/admin/session", "/scores"} {
		if s, _, _ := doH(t, a, "GET", p, "", "", hdr); s != 401 {
			t.Errorf("%s after logout: %d", p, s)
		}
	}
	if s, _, _ := a.do(t, "POST", "/admin/logout", "", ""); s != 204 {
		t.Errorf("logout without cookie: %d", s)
	}
}

func TestOriginCheck(t *testing.T) {
	a := newTestAPI(t)
	ck := cookieHeader(t, a)
	for _, p := range []string{"/admin/login", "/admin/logout", "/scores/increment/1", "/scores/decrement/1", "/reset"} {
		evil := map[string]string{"Origin": "http://evil.example", "Cookie": ck}
		s, _, b := doH(t, a, "POST", p, js, goodLogin, evil)
		if s != 403 || strings.TrimSpace(b) != errBody("Forbidden") {
			t.Errorf("%s evil: %d %s", p, s, b)
		}
		same := map[string]string{"Origin": a.URL, "Cookie": ck}
		if s, _, _ := doH(t, a, "POST", p, js, goodLogin, same); s == 403 {
			t.Errorf("%s same origin: 403", p)
		}
		if s, _, _ := doH(t, a, "POST", p, js, goodLogin, map[string]string{"Cookie": ck}); s == 403 {
			t.Errorf("%s no origin: 403", p)
		}
	}
}

func TestAdminRequiresSession(t *testing.T) {
	a := newTestAPI(t)
	ck := cookieHeader(t, a)
	for _, r := range [][2]string{{"GET", "/scores"}, {"GET", "/scores/"}, {"POST", "/scores/increment/1"},
		{"POST", "/scores/decrement/6"}, {"POST", "/reset"}} {
		expect(t, a, r[0], r[1], "", "", 401, errBody("Unauthorized"))
	}
	for _, r := range [][2]string{{"GET", "/scores"}, {"GET", "/scores/"}, {"POST", "/scores/increment/1"},
		{"POST", "/reset"}} {
		if s, _, b := doH(t, a, r[0], r[1], "", "", map[string]string{"Cookie": ck}); s != 200 {
			t.Errorf("%v: %d %s", r, s, b)
		}
	}
}

func TestNoIsAuthorized(t *testing.T) {
	a := newTestAPI(t)
	if s, _, _ := a.do(t, "GET", "/is_authorized", "", ""); s != 404 {
		t.Error(s)
	}
}
