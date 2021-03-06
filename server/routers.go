package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// ルーティング設定とサーバ立ち上げを行う
func routerInit() {
	http.HandleFunc("/auth/login/", authLogin)
	http.HandleFunc("/auth/logout", authLogout)
	http.HandleFunc("/auth/auth", authAuth)
	http.HandleFunc("/auth/start", authStart)
	http.HandleFunc("/auth/setting", authSetting)
	http.HandleFunc("/auth/setting/password", authSettingPassword)
}

// /auth/login サインインのためログインページを表示する
func authLogin(w http.ResponseWriter, r *http.Request) {
	var filename string = r.URL.Path[len("/auth/login/"):]
	if r.URL.Path == "/auth/login/" {
		filename = "index.html"
	}
	filename = fmt.Sprintf(`static/%s`, filename)

	if b, err := static.ReadFile(filename); err != nil {
		writeResponse(w, r, http.StatusNotFound)
	} else {
		writeResponse(w, r, http.StatusOK)
		w.Write(b)
	}
}

// /auth/logout サインアウトのためセッションをクリアする
func authLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, e := r.Cookie("_auth-proxy"); e == nil {
		cookie.MaxAge = -1 // クッキーをクリアするため、MaxAgeフィールドに-1を指定
		cookie.Path = "/"
		http.SetCookie(w, cookie)
	}

	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
}

// /auth/auth nginxのauth_requestに対応する。レスポンス202または、401を返す
func authAuth(w http.ResponseWriter, r *http.Request) {
	if _, e := parseTokenFromCookie(r); e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
	} else {
		writeResponse(w, r, http.StatusAccepted)
	}
}

// /auth/start id/passwordを取得し、認証処理を実行する
func authStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || len(settings.Users) == 0 {
		writeResponse(w, r, http.StatusNotFound)
	} else if u, e := createUser(r.Body); e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
	} else if e := u.Check(); e != nil {
		writeResponse(w, r, http.StatusBadRequest)
	} else if e := u.CheckUser(); e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
	} else if token, e := createToken(u.Id); e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
	} else {
		t := time.Now().In(time.UTC).AddDate(0, 0, 7)
		cookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: t}

		http.SetCookie(w, cookie)
		writeResponse(w, r, http.StatusOK)
	}
}

func authSetting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeResponse(w, r, http.StatusNotFound)
	} else if u, e := parseTokenFromCookie(r); e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
	} else {
		body, _ := json.Marshal(u)
		writeResponse(w, r, http.StatusOK)
		w.Write(body)
	}
}

func authSettingPassword(w http.ResponseWriter, r *http.Request) {
	var changeType = changeType{}
	if r.Method != http.MethodPost {
		writeResponse(w, r, http.StatusNotFound)
	} else if u, e := parseTokenFromCookie(r); e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
	} else if body, e := ioutil.ReadAll(r.Body); e != nil {
		writeResponse(w, r, http.StatusBadRequest)
	} else if e = json.Unmarshal(body, &changeType); e != nil {
		writeResponse(w, r, http.StatusBadRequest)
	} else if e = changePassword(u.Id, changeType); e != nil {
		writeResponse(w, r, http.StatusOK)
	} else {
		writeResponse(w, r, http.StatusBadRequest)
	}
}

func writeResponse(w http.ResponseWriter, r *http.Request, code int) {
	fmt.Printf("%s: %d:%s\n", time.Now().Format("2006-01-02 15:04:05"), code, r.URL)
	w.WriteHeader(code)
	switch code {
	case http.StatusOK:
		return
	case http.StatusAccepted:
		w.Write([]byte("202 accepted"))
	case http.StatusBadRequest:
		w.Write([]byte("400 bad request"))
	case http.StatusUnauthorized:
		w.Write([]byte("401 unauthorized"))
	case http.StatusNotFound:
		w.Write([]byte("404 page notfound"))
	}
}
