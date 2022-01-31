package main

import (
	"log"
	"net/http"
	"time"
)

// ルーティング設定とサーバ立ち上げを行う
func routerInit(port string) {
	http.HandleFunc("/auth/login/", authLogin)
	http.HandleFunc("/auth/logout", authLogout)
	http.HandleFunc("/auth/auth", authAuth)
	http.HandleFunc("/auth/start", authStart)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

// /auth/login サインインのためログインページを表示する
func authLogin(w http.ResponseWriter, r *http.Request) {

	var url string
	if r.URL.Path == "/auth/login/" {
		url = "static/index.html"
	} else {
		url = `static/` + r.URL.Path[len("/auth/login/"):]
	}
	b, err := static.ReadFile(url)
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte("404 page not found"))
		return
	}

	w.WriteHeader(200)
	w.Write(b)
}

// /auth/logout サインアウトのためセッションをクリアする
func authLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("_auth-proxy")
	if err == nil {
		cookie.MaxAge = -1 // クッキーをクリアするため、MaxAgeフィールドに-1を指定
		cookie.Path = "/"
		http.SetCookie(w, cookie)
	}

	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
}

// /auth/auth nginxのauth_requestに対応する。レスポンス202または、401を返す
func authAuth(w http.ResponseWriter, r *http.Request) {
	cookie, e := r.Cookie("_auth-proxy")
	if e != nil {
		e = parseToken(cookie.Value)
	}

	if e != nil {
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized"))
		return
	}

	w.WriteHeader(202)
	w.Write([]byte("202 Authorized"))
}

// /auth/start id/passwordを取得し、認証処理を実行する
func authStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(404)
		w.Write([]byte("404 page not found"))
		return
	}

	if len(settings.Users) == 0 {
		w.WriteHeader(404)
		w.Write([]byte("404 not found"))
		return
	}

	u, message, e := checkStartBody(r.Body)
	if e != nil {
		w.WriteHeader(401)
		w.Write([]byte(message))
		return
	}

	token, e := createToken(u.Id)
	if e != nil {
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized"))
		return
	}

	t := time.Now().In(time.UTC).AddDate(0, 0, 7)
	cookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: t}

	http.SetCookie(w, cookie)
	w.WriteHeader(200)
}
