package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/multios12/auth-service/setting"
)

// ルーティング設定とサーバ立ち上げを行う
func routerInit() {
	http.HandleFunc("POST /auth/api/login", postAuthApiLogin)
	http.HandleFunc("GET  /auth/api/logout", getAuthApiLogout)
	http.HandleFunc("GET  /auth/api/auth", authApiAuth)
	http.HandleFunc("GET  /auth/api/info", getAuthApiInfo)
	http.HandleFunc("POST /auth/api/info", postAuthApiinfo)

	http.HandleFunc("GET  /auth/{file...}", getAuthHtml)
}

// /auth/*.html 指定されたファイルを返す
func getAuthHtml(w http.ResponseWriter, r *http.Request) {
	filename := path.Join("static", r.PathValue("file"))

	b, err := static.ReadFile(filename)
	if err != nil {
		writeResponse(w, r, http.StatusNotFound)
		return
	}
	writeResponseBody(w, r, http.StatusOK, b)
}

// /auth/api/login id/passwordを取得し、認証処理を実行する
func postAuthApiLogin(w http.ResponseWriter, r *http.Request) {
	if len(setting.Settings.Users) == 0 {
		writeResponse(w, r, http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	u, e := createUser(r.Body)
	if e != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(e, &maxBytesErr) {
			writeResponse(w, r, http.StatusRequestEntityTooLarge)
		} else {
			writeResponse(w, r, http.StatusUnauthorized)
		}
		authLoginLimiter.RecordFailure(loginRateLimitKey(r, ""))
		return
	}

	if e := u.Check(); e != nil {
		authLoginLimiter.RecordFailure(loginRateLimitKey(r, u.Id))
		writeResponse(w, r, http.StatusBadRequest)
		return
	}

	key := loginRateLimitKey(r, u.Id)
	if ok, retryAfter := authLoginLimiter.Allow(key); !ok {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
		writeResponse(w, r, http.StatusTooManyRequests)
		return
	}

	if e := u.CheckUser(); e != nil {
		authLoginLimiter.RecordFailure(key)
		writeResponse(w, r, http.StatusUnauthorized)
		return
	}

	authLoginLimiter.Reset(key)
	if token, e := createToken(u.Id, u.TokenVersion); e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
	} else {
		// トークンをクッキーに保存
		t := time.Now().In(time.UTC).AddDate(0, 0, 7)
		cookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: t, HttpOnly: true, Secure: true}
		http.SetCookie(w, cookie)

		// トークンをレスポンスに返す
		if r.Header.Get("X-Return-Token") == "true" {
			body, _ := json.Marshal(map[string]string{"token": token})
			writeResponseBody(w, r, http.StatusOK, body)
			return
		}
		writeResponse(w, r, http.StatusOK)
	}
}

// /auth/api/logout サインアウトのためセッションをクリアする
func getAuthApiLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, e := r.Cookie("_auth-proxy"); e == nil {
		cookie.MaxAge = -1 // クッキーをクリアするため、MaxAgeフィールドに-1を指定
		cookie.Path = "/"
		cookie.HttpOnly = true
		cookie.SameSite = http.SameSiteLaxMode
		cookie.Secure = true
		http.SetCookie(w, cookie)
	}

	http.Redirect(w, r, "/auth/login.html", http.StatusSeeOther)
}

// /auth/api/auth nginxのauth_requestに対応する。レスポンス202または、401を返す
func authApiAuth(w http.ResponseWriter, r *http.Request) {
	if _, e := parseTokenFromCookie(r); e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
		return
	}
	writeResponse(w, r, http.StatusAccepted)
}

// ----------------------------------------------------------------------------

// GET auth/api/setting 現在のユーザの設定を返す
func getAuthApiInfo(w http.ResponseWriter, r *http.Request) {
	// トークンチェック
	u, e := parseTokenFromCookie(r)
	if e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
		return
	}

	body, _ := json.Marshal(struct {
		Id         string `json:"Id"`
		Permission string `json:"Permission"`
	}{
		Id:         u.Id,
		Permission: u.Permission,
	})
	writeResponseBody(w, r, http.StatusOK, body)
}

// POST auth/api/setting/password パスワードを変更する
func postAuthApiinfo(w http.ResponseWriter, r *http.Request) {
	u, e := parseTokenFromCookie(r)
	if e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var change setting.ChangeType
	if e := json.NewDecoder(r.Body).Decode(&change); e != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(e, &maxBytesErr) {
			writeResponse(w, r, http.StatusRequestEntityTooLarge)
			return
		}
		writeResponse(w, r, http.StatusBadRequest)
		return
	}
	if change.OldPassword == "" || change.NewPassword == "" {
		writeResponse(w, r, http.StatusBadRequest)
		return
	}

	if u.Password == change.OldPassword {
		if e := setting.UpdatePassword(u.Id, change.NewPassword); e != nil {
			writeResponse(w, r, http.StatusInternalServerError)
			return
		}
		writeResponse(w, r, http.StatusOK)
		return
	}

	writeResponse(w, r, http.StatusUnauthorized)
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
	case http.StatusTooManyRequests:
		w.Write([]byte("429 too many requests"))
	case http.StatusRequestEntityTooLarge:
		w.Write([]byte("413 request entity too large"))
	case http.StatusNotFound:
		w.Write([]byte("404 page notfound"))
	case http.StatusInternalServerError:
		w.Write([]byte("500 internal server error"))
	}
}
