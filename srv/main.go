package main

import (
	"crypto/rand"
	"embed"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
)

// ユーザ
type user struct {
	Id         string // ユーザID
	Password   string // パスワード
	Permission string // 権限
}

type authSettings struct {
	Secretkey string // 秘密鍵
	Users     []user // ユーザ情報
}

//go:embed static/*
var static embed.FS       // 静的リソース
var settingsFile *string  // 設定ファイルパス
var settings authSettings //設定

func main() {
	port := flag.String("port", ":3000", "server port")
	settingsFile = flag.String("filename", "./setting.json", "setting file name")
	flag.Parse()

	e := readAuthSettings(*settingsFile)
	if e != nil {
		panic(e)
	}

	routerInit(*port)
}

func readAuthSettings(filename string) error {
	_, e := os.Stat(filename)
	if e != nil {
		settings = authSettings{Secretkey: createRandomString(20), Users: []user{}}
		return writeAuthSettings(filename)
	}

	f, e := ioutil.ReadFile(filename)
	if e == nil {
		e = json.Unmarshal(f, &settings)
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
