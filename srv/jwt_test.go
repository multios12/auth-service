package main

import (
	"bytes"
	"testing"
)

func TestCreateToken(t *testing.T) {
	settings.Secretkey = "0000000000"
	_, err := createToken("test")
	if err != nil {
		t.Error(err)
	}
}

func TestParseToken(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []user{{Id: "test"}}
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
	settings.Users = []user{{Id: "test2"}}
	_, err := parseToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6InRlc3QiLCJuYmYiOjE2NDQ1NDIxMjd9.QUFkNxp5BI-K9pCdMP6l5TDNHPHWRHd4i6SZy99zeOs")
	if err == nil {
		t.Errorf("error")
	}
}

func TestParseToken_tokennotfound(t *testing.T) {
	settings.Secretkey = "0000000000"
	settings.Users = []user{{Id: "test"}}
	_, err := parseToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6InRlc3QiLCJuYmYiOjE2NDQ1NDIxMjd9.QUFkNxp5BI-K9pCdMP6l5TDNHPHWRHd4i6SZy99zeO")
	if err == nil {
		t.Errorf("error")
	}
}

func TestCheckStartBody(t *testing.T) {
	settings.Users = []user{{Id: "test", Password: "test"}}
	var (
		buf = bytes.NewBufferString(`{"Id":"test","Password":"test"}`)
	)
	u, _, err := checkStartBody(buf)
	if err != nil {
		t.Errorf("error")
	}
	if u.Id != "test" {
		t.Errorf("error")
	}
}
func TestCheckStartBody_notfound(t *testing.T) {
	settings.Users = []user{{Id: "test", Password: "test"}}
	var (
		buf = bytes.NewBufferString(`{"Id":"","Password":""}`)
	)
	u, _, err := checkStartBody(buf)
	if err == nil {
		t.Errorf("error")
	}
	if u.Id != "" {
		t.Errorf("error")
	}
}
