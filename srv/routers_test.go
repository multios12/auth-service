package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6InRlc3QiLCJuYmYiOjE2NDQ1NDIxMjd9.QUFkNxp5BI-K9pCdMP6l5TDNHPHWRHd4i6SZy99zeOs"
const errtoken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6InRlc3QiLCJuYmYiOjE2NDQ1NDIxMjd9.QUFkNxp5BI-K9pCdMP6l5TDNHPHWRHd4i6SZy99zeO"

func TestRouterInit(t *testing.T) {
	routerInit()
}

func TestAuthLogin(t *testing.T) {
	b := bytes.NewBufferString(``)
	w, r := createRequestResponse(http.MethodGet, "/auth/login/", b)
	authLogin(w, r)
	if w.Code != http.StatusOK {
		t.Error()
	}

	w, r = createRequestResponse(http.MethodGet, "/auth/notfound", b)
	authLogin(w, r)
	if w.Code != http.StatusNotFound {
		t.Error()
	}

	w, r = createRequestResponse(http.MethodGet, "/auth/login/favicon.ico", b)
	authLogin(w, r)
	if w.Code != http.StatusOK {
		t.Error()
	}
}

func TestAuthLogout(t *testing.T) {
	b := bytes.NewBufferString(``)
	w, r := createRequestResponse(http.MethodGet, "/auth/logout", b)
	authLogout(w, r)
	if w.Code != http.StatusSeeOther {
		t.Error()
	}

	w, r = createRequestResponse(http.MethodGet, "/auth/logout", b)
	ti := time.Now().In(time.UTC).AddDate(0, 0, 7)
	cookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)
	authLogout(w, r)
	if w.Code != http.StatusSeeOther {
		t.Error()
	}
}

func TestAuthAuth(t *testing.T) {
	b := bytes.NewBufferString(``)
	w, r := createRequestResponse(http.MethodGet, "/auth/auth", b)
	ti := time.Now().In(time.UTC).AddDate(0, 0, 7)
	cookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)
	authAuth(w, r)
	if w.Code != http.StatusAccepted {
		t.Error()
	}

	w, r = createRequestResponse(http.MethodGet, "/auth/auth", b)
	cookie = &http.Cookie{Name: "_auth-proxy", Value: errtoken, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)
	authAuth(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Error()
	}
}
func TestAuthStart(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []userType{{Id: "test", Password: "test"}}
	b := bytes.NewBufferString(`{"Id":"test","Password":"test"}`)
	w, r := createRequestResponse(http.MethodPost, "/auth/start", b)
	authStart(w, r)
	if w.Code != http.StatusOK {
		t.Error()
	}
}
func TestAuthStart_MethodError(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []userType{{Id: "test", Password: "test"}}
	b := bytes.NewBufferString(`{"Id":"test","Password":"test"}`)
	w, r := createRequestResponse(http.MethodGet, "/auth/start", b)
	authStart(w, r)
	if w.Code != http.StatusNotFound {
		t.Error()
	}
}
func TestAuthStart_IdError(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []userType{{Id: "test", Password: "test"}}
	b := bytes.NewBufferString(`{"Id":"testnotfound","Password":"test"}`)
	w, r := createRequestResponse(http.MethodGet, "/auth/start", b)
	authStart(w, r)
	if w.Code != http.StatusNotFound {
		t.Error()
	}
}
func TestAuthStart_userError(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []userType{{Id: "test", Password: "test"}}
	b := bytes.NewBufferString(`{"Iest"}`)
	w, r := createRequestResponse(http.MethodPost, "/auth/start", b)
	authStart(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Error()
	}
}
func TestAuthStart_checkError(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []userType{{Id: "test", Password: "test"}}
	b := bytes.NewBufferString(`{"Id":"","Password":""}`)
	w, r := createRequestResponse(http.MethodPost, "/auth/start", b)
	authStart(w, r)
	if w.Code != http.StatusBadRequest {
		t.Error()
	}
}

func TestAuthSetting(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []userType{{Id: "test", Password: "test"}}

	b := bytes.NewBufferString(``)
	w, r := createRequestResponse(http.MethodGet, "/auth/Setting", b)
	createTokenCookie(r, false)
	authSetting(w, r)
	if w.Code != http.StatusOK {
		t.Error()
	}

	w, r = createRequestResponse(http.MethodGet, "/auth/Setting", b)
	createTokenCookie(r, true)
	authSetting(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Error()
	}

	w, r = createRequestResponse(http.MethodPost, "/auth/Setting", b)
	createTokenCookie(r, false)
	authSetting(w, r)
	if w.Code != http.StatusNotFound {
		t.Error()
	}
}

func TestWriteResponse(t *testing.T) {

	w := httptest.NewRecorder()
	r := httptest.NewRequest("get", "/aaa/", nil)
	writeResponse(w, r, 202) //accepted
	writeResponse(w, r, 400) //bad request
	writeResponse(w, r, 401) //unauthorized
	writeResponse(w, r, 404) //notfound
}

func createRequestResponse(method string, target string, body io.Reader) (*httptest.ResponseRecorder, *http.Request) {
	return httptest.NewRecorder(), httptest.NewRequest(method, target, body)
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
