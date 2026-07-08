package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/multios12/auth-service/setting"
)

// ルーティング設定とサーバ立ち上げを行う
func routerInit() {
	if setting.Mode() != 3 {
		http.HandleFunc("POST /auth/api/login", postAuthApiLogin)
	}
	http.HandleFunc("GET  /auth/api/logout", getAuthApiLogout)
	http.HandleFunc("GET  /auth/api/auth", authApiAuth)
	http.HandleFunc("GET  /auth/api/info", getAuthApiInfo)
	http.HandleFunc("POST /auth/api/info", postAuthApiinfo)
	if setting.Mode() == 2 {
		http.HandleFunc("POST /auth/api/passkey/register/options", postAuthApiPasskeyRegisterOptions)
		http.HandleFunc("POST /auth/api/passkey/register/verify", postAuthApiPasskeyRegisterVerify)
	}
	if setting.Mode() == 3 {
		http.HandleFunc("POST /auth/api/passkey/login/options", postAuthApiPasskeyLoginOptions)
		http.HandleFunc("POST /auth/api/passkey/login/verify", postAuthApiPasskeyLoginVerify)
	}

	http.HandleFunc("GET  /auth/{file...}", getAuthHtml)
}

// /auth/*.html 指定されたファイルを返す
func getAuthHtml(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	if file == "login.html" && setting.Mode() == 3 {
		file = "passkey-login.html"
	}
	filename := path.Join("static", file)

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

		next := "/"
		if setting.Mode() == 2 {
			next = "/auth/passkey-register.html"
		}

		// トークンをレスポンスに返す
		if r.Header.Get("X-Return-Token") == "true" {
			body, _ := json.Marshal(map[string]string{"token": token, "next": next})
			writeResponseBody(w, r, http.StatusOK, body)
			return
		}
		if r.Header.Get("Accept") == "application/json" || r.Header.Get("X-Return-Next") == "true" {
			body, _ := json.Marshal(map[string]string{"next": next})
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

// /auth/api/passkey/register/options パスキー登録用のチャレンジを返す
func postAuthApiPasskeyRegisterOptions(w http.ResponseWriter, r *http.Request) {
	u, e := parseTokenFromCookie(r)
	if e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
		return
	}

	if _, e := setting.EnsureUserHandle(u.Id); e != nil {
		writeResponse(w, r, http.StatusInternalServerError)
		return
	}
	u, ok := setting.FindUserByID(u.Id)
	if !ok {
		writeResponse(w, r, http.StatusUnauthorized)
		return
	}

	wa, e := newWebAuthn(r)
	if e != nil {
		writeResponse(w, r, http.StatusInternalServerError)
		return
	}
	creation, session, e := wa.BeginRegistration(
		passkeyUser{user: u},
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithAuthenticatorSelection(discoverableAuthenticatorSelection()),
	)
	if e != nil {
		writeResponse(w, r, http.StatusInternalServerError)
		return
	}

	sessionID, e := createSessionID()
	if e != nil {
		writeResponse(w, r, http.StatusInternalServerError)
		return
	}
	passkeySessions.set(sessionID, passkeySessionEntry{
		Session: *session,
		Kind:    passkeySessionKindRegister,
		UserID:  u.Id,
		Expires: time.Now().Add(passkeySessionTTL),
	})
	setPasskeySessionCookie(w, r, sessionID, passkeySessionTTL)

	body, _ := json.Marshal(creation)
	writeResponseBody(w, r, http.StatusOK, body)
}

// /auth/api/passkey/register/verify パスキー登録結果を検証して保存する
func postAuthApiPasskeyRegisterVerify(w http.ResponseWriter, r *http.Request) {
	u, e := parseTokenFromCookie(r)
	if e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
		return
	}

	cookie, e := r.Cookie(passkeySessionCookieName)
	if e != nil {
		writeResponse(w, r, http.StatusBadRequest)
		return
	}
	entry, ok := passkeySessions.take(cookie.Value, passkeySessionKindRegister)
	clearPasskeySessionCookie(w, r)
	if !ok || entry.UserID != u.Id {
		writeResponse(w, r, http.StatusBadRequest)
		return
	}

	u, ok = setting.FindUserByID(u.Id)
	if !ok {
		writeResponse(w, r, http.StatusUnauthorized)
		return
	}

	wa, e := newWebAuthn(r)
	if e != nil {
		writeResponse(w, r, http.StatusInternalServerError)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPasskeyRequestBodyBytes)
	credential, e := wa.FinishRegistration(passkeyUser{user: u}, entry.Session, r)
	if e != nil {
		writeResponse(w, r, http.StatusBadRequest)
		return
	}

	passkey, e := credentialToPasskey(credential, setting.PasskeyType{})
	if e != nil {
		writeResponse(w, r, http.StatusInternalServerError)
		return
	}
	if e := setting.AddPasskey(u.Id, passkey); e != nil {
		writeResponse(w, r, http.StatusInternalServerError)
		return
	}
	writeResponse(w, r, http.StatusOK)
}

// /auth/api/passkey/login/options パスキー認証用のチャレンジを返す
func postAuthApiPasskeyLoginOptions(w http.ResponseWriter, r *http.Request) {
	wa, e := newWebAuthn(r)
	if e != nil {
		writeResponse(w, r, http.StatusInternalServerError)
		return
	}
	assertion, session, e := wa.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationPreferred))
	if e != nil {
		writeResponse(w, r, http.StatusInternalServerError)
		return
	}

	sessionID, e := createSessionID()
	if e != nil {
		writeResponse(w, r, http.StatusInternalServerError)
		return
	}
	passkeySessions.set(sessionID, passkeySessionEntry{
		Session: *session,
		Kind:    passkeySessionKindLogin,
		Expires: time.Now().Add(passkeySessionTTL),
	})
	setPasskeySessionCookie(w, r, sessionID, passkeySessionTTL)

	body, _ := json.Marshal(assertion)
	writeResponseBody(w, r, http.StatusOK, body)
}

// /auth/api/passkey/login/verify パスキー認証結果を検証してJWT Cookieを発行する
func postAuthApiPasskeyLoginVerify(w http.ResponseWriter, r *http.Request) {
	cookie, e := r.Cookie(passkeySessionCookieName)
	if e != nil {
		writeResponse(w, r, http.StatusBadRequest)
		return
	}
	entry, ok := passkeySessions.take(cookie.Value, passkeySessionKindLogin)
	clearPasskeySessionCookie(w, r)
	if !ok {
		writeResponse(w, r, http.StatusBadRequest)
		return
	}

	wa, e := newWebAuthn(r)
	if e != nil {
		writeResponse(w, r, http.StatusInternalServerError)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPasskeyRequestBodyBytes)
	var loggedInUser setting.UserType
	credential, e := wa.FinishDiscoverableLogin(func(rawID, userHandle []byte) (webauthn.User, error) {
		user, err := discoverableUser(rawID, userHandle)
		if err != nil {
			return nil, err
		}
		if pu, ok := user.(passkeyUser); ok {
			loggedInUser = pu.user
		}
		return user, nil
	}, entry.Session, r)
	if e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
		return
	}
	if loggedInUser.Id == "" {
		writeResponse(w, r, http.StatusUnauthorized)
		return
	}
	credentialID := base64.RawURLEncoding.EncodeToString(credential.ID)
	current := findPasskey(loggedInUser, credentialID)
	passkey, e := credentialToPasskey(credential, current)
	if e != nil {
		writeResponse(w, r, http.StatusInternalServerError)
		return
	}
	if e := setting.UpdatePasskey(credentialID, passkey); e != nil {
		writeResponse(w, r, http.StatusInternalServerError)
		return
	}

	token, e := createToken(loggedInUser.Id, loggedInUser.TokenVersion)
	if e != nil {
		writeResponse(w, r, http.StatusUnauthorized)
		return
	}
	t := time.Now().In(time.UTC).AddDate(0, 0, 7)
	authCookie := &http.Cookie{Name: "_auth-proxy", Value: token, SameSite: http.SameSiteLaxMode, Path: "/", Expires: t, HttpOnly: true, Secure: true}
	http.SetCookie(w, authCookie)
	writeResponse(w, r, http.StatusOK)
}

func setPasskeySessionCookie(w http.ResponseWriter, r *http.Request, sessionID string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     passkeySessionCookieName,
		Value:    sessionID,
		SameSite: http.SameSiteLaxMode,
		Path:     "/auth/api/passkey",
		Expires:  time.Now().Add(ttl),
		HttpOnly: true,
		Secure:   !isLocalHost(hostWithoutPort(requestHost(r))),
	})
}

func clearPasskeySessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     passkeySessionCookieName,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
		Path:     "/auth/api/passkey",
		HttpOnly: true,
		Secure:   !isLocalHost(hostWithoutPort(requestHost(r))),
	})
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
