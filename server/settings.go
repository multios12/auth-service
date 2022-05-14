package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
)

// 設定変更情報
type changeType struct {
	Id          string // ユーザID
	Password    string // パスワード
	NewPassword string // 新しいパスワード
	Permission  string // 権限
}

func readAuthSettings(filename string) error {
	_, e := os.Stat(filename)
	if e != nil {
		settings = settingsType{Secretkey: createRandomString(20), Users: []userType{}}
		return writeAuthSettings(filename)
	}

	f, e := ioutil.ReadFile(filename)
	if e == nil {
		e = json.Unmarshal(f, &settings)
		if e == nil && len(settings.Secretkey) == 0 {
			settings.Secretkey = createRandomString(20)
			return writeAuthSettings(filename)
		}
	}
	return e
}

func writeAuthSettings(filename string) error {
	b, e := json.Marshal(settings)
	if e != nil {
		panic(e)
	}
	return ioutil.WriteFile(filename, b, os.ModePerm)
}

func createRandomString(digit uint32) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, digit)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}

	var result string
	for _, v := range b {
		result += string(letters[int(v)%len(letters)])
	}
	return result
}

func changePassword(id string, c changeType) error {
	for i, v := range settings.Users {
		if v.Id == id {
			if v.Password == c.Password {
				settings.Users[i].Password = c.NewPassword
				return writeAuthSettings(*settingsFile)
			} else {
				return errors.New("")
			}
		}
	}
	return errors.New("")
}
