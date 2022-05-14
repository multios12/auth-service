package main

import (
	"bytes"
	"net/http"
	"testing"
	"time"
)

func TestCreateToken(t *testing.T) {
	settings.Secretkey = "0000000000"
	_, err := createToken("test")
	if err != nil {
		t.Error(err)
	}
}

func TestParseTokenFromCookie(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []userType{{Id: "test"}}

	r, _ := http.NewRequest("GET", "/index.html", bytes.NewBufferString(`{"Id":"test","Password":"test"}`))
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6InRlc3QiLCJuYmYiOjE2NDQ1NDIxMjd9.QUFkNxp5BI-K9pCdMP6l5TDNHPHWRHd4i6SZy99zeOs"
	ti := time.Now().In(time.UTC).AddDate(0, 0, 7)
	cookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)

	u, err := parseTokenFromCookie(r)
	if err != nil {
		t.Error(err)
	}

	if u.Id != "test" {
		t.Errorf("id error")
	}
}

func TestParseTokenFromCookie_cookienotfound(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []userType{{Id: "test"}}

	r, _ := http.NewRequest("GET", "/index.html", bytes.NewBufferString(`{"Id":"test","Password":"test"}`))
	_, err := parseTokenFromCookie(r)
	if err == nil {
		t.Error(err)
	}
}

func TestParseTokenFromCookie_tokenerror(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []userType{{Id: "test"}}

	r, _ := http.NewRequest("GET", "/index.html", bytes.NewBufferString(`{"Id":"test","Password":"test"}`))
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6InRlc3QiLCJuYmYiOjE2NDQ1NDIxMjd9.QUFkNxp5BI-K9pCdMP6l5TDNHPHWRHd4i6SZy99zeO"
	ti := time.Now().In(time.UTC).AddDate(0, 0, 7)
	cookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: ti}
	r.AddCookie(cookie)

	_, err := parseTokenFromCookie(r)
	if err == nil {
		t.Error(err)
	}
}

func TestParseToken(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []userType{{Id: "test"}}
	u, err := parseToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6InRlc3QiLCJuYmYiOjE2NDQ1NDIxMjd9.QUFkNxp5BI-K9pCdMP6l5TDNHPHWRHd4i6SZy99zeOs")
	if err != nil {
		t.Error(err)
	}

	if u.Id != "test" {
		t.Errorf("id error")
	}
}

func TestParseToken_idnotfound(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []userType{{Id: "test2"}}
	_, err := parseToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6InRlc3QiLCJuYmYiOjE2NDQ1NDIxMjd9.QUFkNxp5BI-K9pCdMP6l5TDNHPHWRHd4i6SZy99zeOs")
	if err == nil {
		t.Errorf("error")
	}
}

func TestParseToken_tokennotfound(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []userType{{Id: "test"}}
	_, err := parseToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6InRlc3QiLCJuYmYiOjE2NDQ1NDIxMjd9.QUFkNxp5BI-K9pCdMP6l5TDNHPHWRHd4i6SZy99zeO")
	if err == nil {
		t.Errorf("error")
	}
}

func TestCreateUser(t *testing.T) {
	settings.Users = []userType{{Id: "test", Password: "test"}}
	var (
		buf = bytes.NewBufferString(`{"Id":"test","Password":"test"}`)
	)
	u, err := createUser(buf)
	if err != nil {
		t.Errorf("error")
	}
	if u.Id != "test" {
		t.Errorf("error")
	}
}

func TestCheckStartBody(t *testing.T) {
	m := userType{Id: "test", Password: "test"}.Check()
	if m != nil {
		t.Errorf("error:%s", m)
	}
}

func TestCheckStartBody_notfound(t *testing.T) {
	user := userType{Id: "", Password: ""}.Check()
	if user == nil {
		t.Errorf("error")
	}
}
