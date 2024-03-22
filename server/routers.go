package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/multios12/auth-service/setting"
)

// ルーティング設定とサーバ立ち上げを行う
func routerInit() {
	http.HandleFunc("/auth/api/login", postAuthApiLogin)
	http.HandleFunc("/auth/api/logout", getAuthApiLogout)
	http.HandleFunc("/auth/api/auth", authApiAuth)
	http.HandleFunc("/auth/api/info", authApiInfo)
	// http.HandleFunc("GET  /auth/api/info", getAuthApiInfo)
	// http.HandleFunc("POST /auth/api/info", postAuthApiinfo)

	// TODO:ユーザ管理画面の実装
	// http.HandleFunc("/auth/api/users", authUsers)
	http.HandleFunc("/auth/", getAuthHtml)
}

// /auth/*.html 指定されたファイルを返す
func getAuthHtml(w http.ResponseWriter, r *http.Request) {
	var filename string = r.URL.Path[len("/auth/"):]
	filename = path.Join("static", filename)

	if b, err := static.ReadFile(filename); err != nil {
		writeResponse(w, r, http.StatusNotFound)
	} else {
		writeResponseBody(w, r, http.StatusOK, b)
	}
}

// /auth/api/login id/passwordを取得し、認証処理を実行する
func postAuthApiLogin(w http.ResponseWriter, r *http.Request) {
	if len(setting.Settings.Users) == 0 {
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

// /auth/api/logout サインアウトのためセッションをクリアする
func getAuthApiLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, e := r.Cookie("_auth-proxy"); e == nil {
		cookie.MaxAge = -1 // クッキーをクリアするため、MaxAgeフィールドに-1を指定
		cookie.Path = "/"
		http.SetCookie(w, cookie)
	}

	http.Redirect(w, r, "/auth/login.html", http.StatusSeeOther)
}

// /auth/api/auth nginxのauth_requestに対応する。レスポンス202または、401を返す
func authApiAuth(w http.ResponseWriter, r *http.Request) {
	if _, e := parseTokenFromCookie(r); e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
	} else {
		writeResponse(w, r, http.StatusAccepted)
	}
}

// ----------------------------------------------------------------------------
func authApiInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		postAuthApiinfo(w, r)
	} else {
		getAuthApiInfo(w, r)
	}
}

// GET auth/api/setting 現在のユーザの設定を返す
func getAuthApiInfo(w http.ResponseWriter, r *http.Request) {
	// トークンチェック
	u, e := parseTokenFromCookie(r)
	if e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		u.Password = ""
		body, _ := json.Marshal(u)
		writeResponseBody(w, r, http.StatusOK, body)
	}
}

// POST auth/api/setting/password パスワードを変更する
func postAuthApiinfo(w http.ResponseWriter, r *http.Request) {
	u, e := parseTokenFromCookie(r)
	if e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
		return
	}

	bytes, _ := io.ReadAll(r.Body)
	var change setting.ChangeType
	json.Unmarshal(bytes, &change)

	if u.Password == change.OldPassword {
		setting.UpdatePassword(u.Id, change.NewPassword)
	}
}

// ----------------------------------------------------------------------------

// レスポンスコード及びボディを書き込む
func writeResponseBody(w http.ResponseWriter, r *http.Request, code int, body []byte) {
	writeResponse(w, r, code)
	w.Write(body)
}

// レスポンスコードを書き込む
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
