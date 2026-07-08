package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/multios12/auth-service/setting"
	"golang.org/x/net/publicsuffix"
)

const passkeySessionCookieName = "_auth-passkey-session" // パスキー認証セッションCookie名

const passkeySessionTTL = 5 * time.Minute // パスキーチャレンジ有効期間

const maxPasskeyRequestBodyBytes int64 = 65536 // パスキー検証リクエスト本文サイズ

// パスキーセッションの種類
type passkeySessionKind string

const (
	passkeySessionKindRegister passkeySessionKind = "register" // 登録セッション
	passkeySessionKindLogin    passkeySessionKind = "login"    // 認証セッション
)

// パスキーセッション情報
type passkeySessionEntry struct {
	Session webauthn.SessionData // WebAuthnセッション情報
	Kind    passkeySessionKind   // セッション種別
	UserID  string               // 登録対象ユーザID
	Expires time.Time            // 有効期限
}

// パスキーセッションストア
type passkeySessionStore struct {
	mu       sync.Mutex                     // 状態保護用の排他制御
	sessions map[string]passkeySessionEntry // セッションIDごとの状態
}

// WebAuthnユーザアダプタ
type passkeyUser struct {
	user setting.UserType // 設定上のユーザ情報
}

var passkeySessions = &passkeySessionStore{
	sessions: map[string]passkeySessionEntry{},
}

func (s *passkeySessionStore) set(id string, entry passkeySessionEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()
	s.sessions[id] = entry
}

func (s *passkeySessionStore) take(id string, kind passkeySessionKind) (passkeySessionEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()
	entry, ok := s.sessions[id]
	if !ok || entry.Kind != kind || time.Now().After(entry.Expires) {
		delete(s.sessions, id)
		return passkeySessionEntry{}, false
	}

	delete(s.sessions, id)
	return entry, true
}

func (s *passkeySessionStore) cleanupLocked() {
	now := time.Now()
	for id, entry := range s.sessions {
		if now.After(entry.Expires) {
			delete(s.sessions, id)
		}
	}
}

// ユーザハンドルを返す
func (u passkeyUser) WebAuthnID() []byte {
	return []byte(u.user.UserHandle)
}

// ユーザ名を返す
func (u passkeyUser) WebAuthnName() string {
	return u.user.Id
}

// 表示名を返す
func (u passkeyUser) WebAuthnDisplayName() string {
	return u.user.Id
}

// アイコンURLを返す
func (u passkeyUser) WebAuthnIcon() string {
	return ""
}

// 登録済み認証情報を返す
func (u passkeyUser) WebAuthnCredentials() []webauthn.Credential {
	credentials := make([]webauthn.Credential, 0, len(u.user.Passkeys))
	for _, passkey := range u.user.Passkeys {
		credential, err := passkeyToCredential(passkey)
		if err == nil {
			credentials = append(credentials, credential)
		}
	}
	return credentials
}

func newWebAuthn(r *http.Request) (*webauthn.WebAuthn, error) {
	host := requestHost(r)
	rpID := relyingPartyID(host)
	origin := requestOrigin(r, host)
	return webauthn.New(&webauthn.Config{
		RPID:                   rpID,
		RPDisplayName:          rpID,
		RPOrigins:              []string{origin},
		AttestationPreference:  protocol.PreferNoAttestation,
		AuthenticatorSelection: discoverableAuthenticatorSelection(),
	})
}

func requestHost(r *http.Request) string {
	host := strings.TrimSpace(r.Host)
	if host == "" {
		host = strings.TrimSpace(r.URL.Host)
	}
	if host == "" {
		return "localhost"
	}
	return host
}

func requestOrigin(r *http.Request, host string) string {
	scheme := "https"
	if isLocalHost(hostWithoutPort(host)) {
		scheme = "http"
	} else if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + host
}

func relyingPartyID(host string) string {
	host = hostWithoutPort(host)
	if host == "" {
		return "localhost"
	}
	if isLocalHost(host) || net.ParseIP(host) != nil {
		return host
	}
	domain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil || domain == "" {
		return host
	}
	return domain
}

func hostWithoutPort(host string) string {
	if strings.HasPrefix(host, "[") {
		if h, _, err := net.SplitHostPort(host); err == nil {
			return strings.Trim(h, "[]")
		}
		return strings.Trim(host, "[]")
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

func isLocalHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func discoverableAuthenticatorSelection() protocol.AuthenticatorSelection {
	return protocol.AuthenticatorSelection{
		RequireResidentKey: protocol.ResidentKeyRequired(),
		ResidentKey:        protocol.ResidentKeyRequirementRequired,
		UserVerification:   protocol.VerificationPreferred,
	}
}

func createSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func passkeyToCredential(passkey setting.PasskeyType) (webauthn.Credential, error) {
	if passkey.CredentialData != "" {
		b, err := base64.RawURLEncoding.DecodeString(passkey.CredentialData)
		if err != nil {
			return webauthn.Credential{}, err
		}
		var credential webauthn.Credential
		if err := json.Unmarshal(b, &credential); err != nil {
			return webauthn.Credential{}, err
		}
		return credential, nil
	}

	id, err := base64.RawURLEncoding.DecodeString(passkey.CredentialID)
	if err != nil {
		return webauthn.Credential{}, err
	}
	publicKey, err := base64.RawURLEncoding.DecodeString(passkey.PublicKey)
	if err != nil {
		return webauthn.Credential{}, err
	}
	aaguid, _ := base64.RawURLEncoding.DecodeString(passkey.AAGUID)
	return webauthn.Credential{
		ID:        id,
		PublicKey: publicKey,
		Authenticator: webauthn.Authenticator{
			AAGUID:    aaguid,
			SignCount: passkey.SignCount,
		},
	}, nil
}

func credentialToPasskey(credential *webauthn.Credential, current setting.PasskeyType) (setting.PasskeyType, error) {
	if credential == nil {
		return setting.PasskeyType{}, fmt.Errorf("credential is nil")
	}
	b, err := json.Marshal(credential)
	if err != nil {
		return setting.PasskeyType{}, err
	}
	return setting.PasskeyType{
		CredentialID:   base64.RawURLEncoding.EncodeToString(credential.ID),
		PublicKey:      base64.RawURLEncoding.EncodeToString(credential.PublicKey),
		SignCount:      credential.Authenticator.SignCount,
		AAGUID:         base64.RawURLEncoding.EncodeToString(credential.Authenticator.AAGUID),
		Name:           current.Name,
		CreatedAt:      current.CreatedAt,
		CredentialData: base64.RawURLEncoding.EncodeToString(b),
	}, nil
}

func findPasskey(user setting.UserType, credentialID string) setting.PasskeyType {
	for _, passkey := range user.Passkeys {
		if passkey.CredentialID == credentialID {
			return passkey
		}
	}
	return setting.PasskeyType{}
}

func discoverableUser(rawID, userHandle []byte) (webauthn.User, error) {
	handle := string(userHandle)
	if handle != "" {
		if user, ok := setting.FindUserByHandle(handle); ok {
			return passkeyUser{user: user}, nil
		}
	}

	credentialID := base64.RawURLEncoding.EncodeToString(rawID)
	if user, ok := setting.FindUserByCredentialID(credentialID); ok {
		return passkeyUser{user: user}, nil
	}
	return nil, fmt.Errorf("passkey notfound")
}
