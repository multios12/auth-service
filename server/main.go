package main

import (
	"embed"
	"flag"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/multios12/auth-service/setting"
)

//go:embed static/*
var static embed.FS // 静的リソース

func main() {
	port := flag.String("port", ":3000", "server port")
	filename := flag.String("filename", "./setting.json", "setting file name")
	*filename, _ = filepath.Abs(*filename)
	flag.Parse()

	fmt.Println("Start: auth-service")

	if e := setting.Read(*filename); e != nil {
		panic(e)
	}

	routerInit()
	if e := http.ListenAndServe(*port, nil); e != nil {
		panic(e)
	}
}
