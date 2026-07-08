package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/multios12/auth-service/setting"
)

var token string
var errtoken string

func TestMain(m *testing.M) {
	// 初期化処理
	f, _ := os.CreateTemp("", "auth-service-test-setting")
	filename := f.Name()
	f.Write([]byte(`{"Secretkey":"0000000000","Users":[{"Id":"test","Password":"test","Permission":""}]}`))
	f.Close()
	setting.Read(filename)
	token, _ = createToken("test")
	errtoken = token + "x"

	code := m.Run()

	// ここでテストのお片づけ
	os.Remove(filename)
	os.Exit(code)
}

func TestRouterInit(t *testing.T) {
	origMux := http.DefaultServeMux
	origMode := setting.Settings.Mode
	t.Cleanup(func() {
		http.DefaultServeMux = origMux
		setting.Settings.Mode = origMode
	})

	tests := []struct {
		mode             int
		passwordWant     int
		registerWant     int
		passkeyLoginWant int
	}{
		{mode: 1, passwordWant: http.StatusOK, registerWant: http.StatusMethodNotAllowed, passkeyLoginWant: http.StatusMethodNotAllowed},
		{mode: 2, passwordWant: http.StatusOK, registerWant: http.StatusUnauthorized, passkeyLoginWant: http.StatusMethodNotAllowed},
		{mode: 3, passwordWant: http.StatusMethodNotAllowed, registerWant: http.StatusMethodNotAllowed, passkeyLoginWant: http.StatusOK},
	}

	for _, tt := range tests {
		http.DefaultServeMux = http.NewServeMux()
		setting.Settings.Mode = tt.mode
		routerInit()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/auth/api/login", strings.NewReader(`{"Id":"test","Password":"test"}`))
		http.DefaultServeMux.ServeHTTP(w, r)
		if w.Code != tt.passwordWant {
			t.Fatalf("mode %d password login: got %d, want %d", tt.mode, w.Code, tt.passwordWant)
		}

		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodPost, "/auth/api/passkey/register/options", nil)
		http.DefaultServeMux.ServeHTTP(w, r)
		if w.Code != tt.registerWant {
			t.Fatalf("mode %d register: got %d, want %d", tt.mode, w.Code, tt.registerWant)
		}

		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodPost, "/auth/api/passkey/login/options", nil)
		http.DefaultServeMux.ServeHTTP(w, r)
		if w.Code != tt.passkeyLoginWant {
			t.Fatalf("mode %d passkey login: got %d, want %d", tt.mode, w.Code, tt.passkeyLoginWant)
		}
	}
}
func TestGetAuthHtml(t *testing.T) {
	origMode := setting.Settings.Mode
	t.Cleanup(func() { setting.Settings.Mode = origMode })

	setting.Settings.Mode = 1
	w, r := createRequestResponse(http.MethodGet, "/auth/login.html", ``)
	r.SetPathValue("file", "login.html")
	getAuthHtml(w, r)
	if w.Code != http.StatusOK {
		t.Error()
	}
	if !strings.Contains(w.Body.String(), "<div class=\"title\">login</div>") {
		t.Error("login html not returned")
	}

	setting.Settings.Mode = 3
	w, r = createRequestResponse(http.MethodGet, "/auth/login.html", ``)
	r.SetPathValue("file", "login.html")
	getAuthHtml(w, r)
	if w.Code != http.StatusOK {
		t.Error()
	}
	if !strings.Contains(w.Body.String(), "passkey login") {
		t.Error("passkey login html not returned")
	}

	w, r = createRequestResponse(http.MethodGet, "/auth/notfound", ``)
	r.SetPathValue("file", "notfound")
	getAuthHtml(w, r)
	if w.Code != http.StatusNotFound {
		t.Error()
	}
}

func TestGetAuthApiLogout(t *testing.T) {
	w, r := createRequestResponse(http.MethodGet, "/auth/api/logout", ``)
	getAuthApiLogout(w, r)
	if w.Code != http.StatusSeeOther {
		t.Error()
	}

	w, r = createRequestResponse(http.MethodGet, "/auth/api/logout", ``)
	ti := time.Now().In(time.UTC).AddDate(0, 0, 7)
	cookie := &http.Cookie{Name: "_auth-proxy", Value: errtoken, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)
	getAuthApiLogout(w, r)
	if w.Code != http.StatusSeeOther {
		t.Error()
	}
}

func TestAuthApiAuth(t *testing.T) {
	w, r := createRequestResponse(http.MethodGet, "/auth/api/auth", ``)
	ti := time.Now().In(time.UTC).AddDate(0, 0, 7)
	cookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)
	authApiAuth(w, r)
	if w.Code != http.StatusAccepted {
		t.Error()
	}

	w, r = createRequestResponse(http.MethodGet, "/auth/api/auth", ``)
	cookie = &http.Cookie{Name: "_auth-proxy", Value: errtoken, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)
	authApiAuth(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Error()
	}
}
func TestPostAuthLogin(t *testing.T) {
	w, r := createRequestResponse(http.MethodPost, "/auth/api/login", `{"Id":"test","Password":"test"}`)
	postAuthApiLogin(w, r)
	if w.Code != http.StatusOK {
		t.Error()
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly {
		t.Error()
	}
	if !cookies[0].Secure {
		t.Error()
	}
}

func TestPostAuthLogin_ReturnToken(t *testing.T) {
	w, r := createRequestResponse(http.MethodPost, "/auth/api/login", `{"Id":"test","Password":"test"}`)
	r.Header.Set("X-Return-Token", "true")
	postAuthApiLogin(w, r)
	if w.Code != http.StatusOK {
		t.Error()
	}

	if got := w.Body.String(); got == "" {
		t.Error()
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure {
		t.Error()
	}
}

func TestPostAuthLogin_ReturnNext(t *testing.T) {
	origMode := setting.Settings.Mode
	t.Cleanup(func() { setting.Settings.Mode = origMode })
	setting.Settings.Mode = 2

	w, r := createRequestResponse(http.MethodPost, "/auth/api/login", `{"Id":"test","Password":"test"}`)
	r.Header.Set("X-Return-Next", "true")
	postAuthApiLogin(w, r)
	if w.Code != http.StatusOK {
		t.Error()
	}

	var body map[string]string
	if e := json.Unmarshal(w.Body.Bytes(), &body); e != nil {
		t.Fatal(e)
	}
	if body["next"] != "/auth/passkey-register.html" {
		t.Fatalf("got %q", body["next"])
	}
}

func TestAuthStart_IdError(t *testing.T) {
	w, r := createRequestResponse(http.MethodGet, "/auth/api/login", `{"Id":"testnotfound","Password":"test"}`)
	postAuthApiLogin(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Error()
	}
}
func TestAuthStart_userError(t *testing.T) {
	w, r := createRequestResponse(http.MethodPost, "/auth/api/login", `{"Iest"}`)
	postAuthApiLogin(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Error()
	}
}
func TestAuthStart_checkError(t *testing.T) {
	w, r := createRequestResponse(http.MethodPost, "/auth/api/login", `{"Id":"","Password":""}`)
	postAuthApiLogin(w, r)
	if w.Code != http.StatusBadRequest {
		t.Error()
	}
}

func TestGetAuthApiInfo(t *testing.T) {
	w, r := createRequestResponse(http.MethodGet, "/auth/api/info", ``)
	ti := time.Now().In(time.UTC).AddDate(0, 0, 7)
	cookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)
	getAuthApiInfo(w, r)
	if w.Code != http.StatusOK {
		t.Error()
		return
	}

	w, r = createRequestResponse(http.MethodGet, "/auth/api/info", `{"Id":"test","Password":"error"}`)
	ti = time.Now().In(time.UTC).AddDate(0, 0, 7)
	cookie = &http.Cookie{Name: "_auth-proxy", Value: errtoken, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)
	getAuthApiInfo(w, r)
	if w.Code == http.StatusOK {
		t.Error()
	}
}
func TestPostAuthApiInfo(t *testing.T) {
	origSecret := setting.Settings.Secretkey
	origUsers := append([]setting.UserType(nil), setting.Settings.Users...)
	t.Cleanup(func() {
		users := append([]setting.UserType(nil), origUsers...)
		setting.Settings = setting.SettingsType{Secretkey: origSecret, Users: users}
	})

	w, r := createRequestResponse(http.MethodPost, "/auth/api/info", `{"OldPassword":"test","NewPassword":"test"}`)
	ti := time.Now().In(time.UTC).AddDate(0, 0, 7)
	cookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)
	postAuthApiinfo(w, r)
	if w.Code != http.StatusOK {
		t.Error()
		return
	}
}

func TestPostAuthLoginRateLimited(t *testing.T) {
	key := loginRateLimitKey(&http.Request{Header: http.Header{}, RemoteAddr: "203.0.113.10:1234"}, "test")
	authLoginLimiter.Reset(key)
	t.Cleanup(func() { authLoginLimiter.Reset(key) })

	for i := 0; i < loginFailureLimit; i++ {
		w, r := createRequestResponse(http.MethodPost, "/auth/api/login", `{"Id":"test","Password":"wrong"}`)
		r.RemoteAddr = "203.0.113.10:1234"
		postAuthApiLogin(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: got %d, want %d", i+1, w.Code, http.StatusUnauthorized)
		}
	}

	w, r := createRequestResponse(http.MethodPost, "/auth/api/login", `{"Id":"test","Password":"wrong"}`)
	r.RemoteAddr = "203.0.113.10:1234"
	postAuthApiLogin(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestPostAuthLoginBodyTooLarge(t *testing.T) {
	bigBody := `{"Id":"test","Password":"` + strings.Repeat("x", int(maxRequestBodyBytes)) + `"}`
	w, r := createRequestResponse(http.MethodPost, "/auth/api/login", bigBody)
	postAuthApiLogin(w, r)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestAuthApiAuthRejectsMalformedAuthorization(t *testing.T) {
	w, r := createRequestResponse(http.MethodGet, "/auth/api/auth", ``)
	r.Header.Set("Authorization", "Basic abc")
	ti := time.Now().In(time.UTC).AddDate(0, 0, 7)
	cookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)
	authApiAuth(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestPostAuthApiInfoBadRequest(t *testing.T) {
	w, r := createRequestResponse(http.MethodPost, "/auth/api/info", `{"OldPassword":"test"`)
	createTokenCookie(r, false)
	postAuthApiinfo(w, r)
	if w.Code != http.StatusBadRequest {
		t.Error()
	}

	w, r = createRequestResponse(http.MethodPost, "/auth/api/info", `{"OldPassword":"test","NewPassword":""}`)
	createTokenCookie(r, false)
	postAuthApiinfo(w, r)
	if w.Code != http.StatusBadRequest {
		t.Error()
	}
}

func TestPostAuthApiInfoUnauthorized(t *testing.T) {
	w, r := createRequestResponse(http.MethodPost, "/auth/api/info", `{"OldPassword":"error","NewPassword":"test2"}`)
	createTokenCookie(r, false)
	postAuthApiinfo(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Error()
	}
}

func TestPostAuthApiPasskeyRegisterOptions(t *testing.T) {
	w, r := createRequestResponse(http.MethodPost, "/auth/api/passkey/register/options", ``)
	createTokenCookie(r, false)
	postAuthApiPasskeyRegisterOptions(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", w.Code, http.StatusOK)
	}
	if len(w.Result().Cookies()) == 0 {
		t.Fatal("session cookie not set")
	}

	var body map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &body); e != nil {
		t.Fatal(e)
	}
	if _, ok := body["publicKey"]; !ok {
		t.Fatal("publicKey notfound")
	}
}

func TestPostAuthApiPasskeyLoginOptions(t *testing.T) {
	w, r := createRequestResponse(http.MethodPost, "/auth/api/passkey/login/options", ``)
	postAuthApiPasskeyLoginOptions(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", w.Code, http.StatusOK)
	}
	if len(w.Result().Cookies()) == 0 {
		t.Fatal("session cookie not set")
	}

	var body map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &body); e != nil {
		t.Fatal(e)
	}
	if _, ok := body["publicKey"]; !ok {
		t.Fatal("publicKey notfound")
	}
}

// ----------------------------------------------------------------------------

func TestWriteResponse(t *testing.T) {

	w := httptest.NewRecorder()
	r := httptest.NewRequest("get", "/aaa/", nil)
	writeResponse(w, r, 202) //accepted
	writeResponse(w, r, 400) //bad request
	writeResponse(w, r, 401) //unauthorized
	writeResponse(w, r, 404) //notfound
}

func createRequestResponse(method string, target string, body string) (*httptest.ResponseRecorder, *http.Request) {
	b := bytes.NewBufferString(body)
	return httptest.NewRecorder(), httptest.NewRequest(method, target, b)
}

func createTokenCookie(r *http.Request, isError bool) {
	ti := time.Now().In(time.UTC).AddDate(0, 0, 7)
	if isError {
		cookie := &http.Cookie{Name: "_auth-proxy", Value: errtoken, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
		r.AddCookie(cookie)
	} else {
		cookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
		r.AddCookie(cookie)
	}
}
