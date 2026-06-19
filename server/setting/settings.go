package setting

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
)

// 設定ファイルを読み込む
func Read(filename string) error {
	settingsFile = filename

	_, e := os.Stat(filename)
	if e != nil {
		Settings = SettingsType{Secretkey: createRandomString(20), Users: []UserType{}}
		fmt.Printf("Load: setting file[%s]\n", settingsFile)
		return Write()
	}

	f, e := os.ReadFile(filename)
	if e == nil {
		e = json.Unmarshal(f, &Settings)
		if e == nil && len(Settings.Secretkey) == 0 {
			Settings.Secretkey = createRandomString(20)
			return Write()
		}
	}
	return e
}

// 設定ファイルを書き込む
func Write() error {
	b, _ := json.Marshal(Settings)
	return os.WriteFile(settingsFile, b, 0o600)
}

// 指定ユーザのパスワードを更新する
func UpdatePassword(id string, password string) error {
	for i := range Settings.Users {
		if Settings.Users[i].Id == id {
			oldPassword := Settings.Users[i].Password
			oldTokenVersion := Settings.Users[i].TokenVersion
			Settings.Users[i].Password = password
			Settings.Users[i].TokenVersion++
			if e := Write(); e != nil {
				Settings.Users[i].Password = oldPassword
				Settings.Users[i].TokenVersion = oldTokenVersion
				return e
			}
			return nil
		}
	}
	return fmt.Errorf("user notfound")
}

// 指定長のランダム文字列を生成する
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
