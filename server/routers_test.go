package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/multios12/auth-service/setting"
)

const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6InRlc3QiLCJuYmYiOjE2NDQ1NDIxMjd9.QUFkNxp5BI-K9pCdMP6l5TDNHPHWRHd4i6SZy99zeOs"
const errtoken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6InRlc3QiLCJuYmYiOjE2NDQ1NDIxMjd9.QUFkNxp5BI-K9pCdMP6l5TDNHPHWRHd4i6SZy99zeO"

func TestMain(m *testing.M) {
	// 初期化処理
	setting.Settings.Secretkey = "0000000000"
	setting.Settings.Users = []setting.UserType{{Id: "test", Password: "test"}}

	code := m.Run()

	// ここでテストのお片づけ
	os.Exit(code)
}

func TestRouterInit(t *testing.T) {
	routerInit()
}
func TestGetAuthHtml(t *testing.T) {
	w, r := createRequestResponse(http.MethodGet, "/auth/.dummy", ``)
	getAuthHtml(w, r)
	if w.Code != http.StatusOK {
		t.Error()
	}

	w, r = createRequestResponse(http.MethodGet, "/auth/notfound", ``)
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
	authApiInfo(w, r)
	if w.Code != http.StatusOK {
		t.Error()
		return
	}

	w, r = createRequestResponse(http.MethodGet, "/auth/api/info", `{"Id":"test","Password":"error"}`)
	ti = time.Now().In(time.UTC).AddDate(0, 0, 7)
	cookie = &http.Cookie{Name: "_auth-proxy", Value: errtoken, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)
	authApiInfo(w, r)
	if w.Code == http.StatusOK {
		t.Error()
	}
}
func TestPostAuthApiInfo(t *testing.T) {
	w, r := createRequestResponse(http.MethodGet, "/auth/api/info", `{"OldPassword":"test","NewPassword":"test"}`)
	ti := time.Now().In(time.UTC).AddDate(0, 0, 7)
	cookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)
	authApiInfo(w, r)
	if w.Code != http.StatusOK {
		t.Error()
		return
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
