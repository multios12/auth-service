package setting

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
)

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

func Write() error {
	b, _ := json.Marshal(Settings)
	return os.WriteFile(settingsFile, b, os.ModePerm)
}

func UpdatePassword(id string, password string) {
	for i, _ := range Settings.Users {
		if Settings.Users[i].Id == id {
			Settings.Users[i].Password = password
			Write()
			break
		}
	}
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
