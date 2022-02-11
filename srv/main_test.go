package main

import "testing"

func TestReadAuthSettings(t *testing.T) {
	e := readAuthSettings("../testdata/setting.json")
	if e != nil {
		t.Error(e)
	}
}

func TestCreateRandomString(t *testing.T) {
	r := createRandomString(20)
	if len(r) != 20 {
		t.Error("error")
	}
}
