package main

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const maxRequestBodyBytes int64 = 4096 // 受付可能最大リクエスト本文サイズ

const (
	loginFailureLimit  = 5                // ログイン失敗回数
	loginFailureWindow = 10 * time.Minute // ログイン失敗ウィンドウ
	loginBlockDuration = 15 * time.Minute // ログインブロック期間の間隔
)

// ログイン試行の状態
type loginAttemptState struct {
	WindowEnds   time.Time // 集計対象の終了時刻
	BlockedUntil time.Time // アクセス停止の終了時刻
	Failures     int       // 失敗回数
}

// ログイン試行を制御する
type loginRateLimiter struct {
	mu      sync.Mutex                    // 状態保護用の排他制御
	records map[string]*loginAttemptState // キーごとの失敗状態
}

var authLoginLimiter = &loginRateLimiter{
	records: map[string]*loginAttemptState{},
}

func (l *loginRateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	state := l.records[key]
	if state == nil {
		return true, 0
	}

	if !state.BlockedUntil.IsZero() && now.Before(state.BlockedUntil) {
		return false, state.BlockedUntil.Sub(now)
	}

	if !state.WindowEnds.IsZero() && now.After(state.WindowEnds) {
		delete(l.records, key)
		return true, 0
	}

	return true, 0
}

func (l *loginRateLimiter) RecordFailure(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	state := l.records[key]
	if state == nil || (!state.WindowEnds.IsZero() && now.After(state.WindowEnds)) {
		state = &loginAttemptState{WindowEnds: now.Add(loginFailureWindow)}
		l.records[key] = state
	}

	state.Failures++
	if state.Failures >= loginFailureLimit {
		state.BlockedUntil = now.Add(loginBlockDuration)
		state.Failures = 0
		state.WindowEnds = state.BlockedUntil
		return false, loginBlockDuration
	}

	return true, 0
}

func (l *loginRateLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.records, key)
}

func loginRateLimitKey(r *http.Request, userID string) string {
	ip := clientIP(r)
	if userID == "" {
		return ip
	}
	return ip + "|" + userID
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if parts := strings.Split(xff, ","); len(parts) > 0 {
			if ip := strings.TrimSpace(parts[0]); ip != "" {
				return ip
			}
		}
	}

	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}
