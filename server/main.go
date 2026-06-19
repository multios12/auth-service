package main

import (
	"flag"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/multios12/auth-service/setting"
)

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
	server := &http.Server{
		Addr:              *port,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if e := server.ListenAndServe(); e != nil {
		panic(e)
	}
}
