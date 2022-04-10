package main

import (
	"embed"
	"flag"
	"net/http"
)

// ユーザ
type userType struct {
	Id         string // ユーザID
	Password   string // パスワード
	Permission string // 権限
}

// 設定
type settingsType struct {
	Secretkey string     // 秘密鍵
	Users     []userType // ユーザ情報
}

//go:embed static/*
var static embed.FS       // 静的リソース
var settingsFile *string  // 設定ファイルパス
var settings settingsType //設定

func main() {
	port := flag.String("port", ":3000", "server port")
	settingsFile = flag.String("filename", "./setting.json", "setting file name")
	flag.Parse()

	if e := readAuthSettings(*settingsFile); e != nil {
		panic(e)
	}

	routerInit()
	if e := http.ListenAndServe(*port, nil); e != nil {
		panic(e)
	}
}
