package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/multios12/auth-service/setting"
)

// トークンを作成する
func createToken(userId string, tokenVersion ...int) (string, error) {
	version := 0
	if len(tokenVersion) > 0 {
		version = tokenVersion[0]
	}
	now := time.Now().In(time.UTC)
	claims := jwt.MapClaims{
		"id":  userId,
		"nbf": now.Unix(),
		"exp": now.AddDate(0, 0, 7).Unix(),
		"ver": version,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(setting.Settings.Secretkey))
}

// リクエストのトークンから、ユーザ情報を取得
func parseTokenFromCookie(r *http.Request) (setting.UserType, error) {
	if authValues, ok := r.Header["Authorization"]; ok && len(authValues) > 0 {
		tokenString := parseBearerToken(authValues[0])
		if len(tokenString) == 0 {
			return setting.UserType{}, fmt.Errorf("invalid authorization header")
		}
		return parseToken(tokenString)
	}

	cookie, err := r.Cookie("_auth-proxy")
	if err != nil {
		return setting.UserType{}, err
	}

	u, err := parseToken(cookie.Value)
	if err != nil {
		return setting.UserType{}, err
	}
	return u, nil
}

// Authorization ヘッダから Bearer トークンを取り出す
func parseBearerToken(header string) string {
	if len(header) == 0 {
		return ""
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

// トークンからユーザ情報を取得
func parseToken(tokenString string) (setting.UserType, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(setting.Settings.Secretkey), nil
	})
	if err != nil {
		return setting.UserType{}, err
	}
	if token == nil {
		return setting.UserType{}, fmt.Errorf("invalid token")
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if _, ok := claims["exp"]; !ok {
			return setting.UserType{}, fmt.Errorf("exp notfound")
		}

		id, ok := claims["id"].(string)
		if !ok {
			return setting.UserType{}, fmt.Errorf("ID notfound")
		}
		tokenVersion, err := claimInt(claims, "ver")
		if err != nil {
			return setting.UserType{}, err
		}

		for _, u := range setting.Settings.Users {
			if u.Id == id {
				if u.TokenVersion != tokenVersion {
					return setting.UserType{}, fmt.Errorf("token expired")
				}
				return u, nil
			}
		}

		return setting.UserType{}, fmt.Errorf("ID notfound")
	}

	return setting.UserType{}, fmt.Errorf("invalid token")
}

func claimInt(claims jwt.MapClaims, key string) (int, error) {
	value, ok := claims[key]
	if !ok {
		return 0, nil
	}

	switch v := value.(type) {
	case float64:
		return int(v), nil
	case float32:
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, err
		}
		return int(i), nil
	default:
		return 0, errors.New("invalid token claim")
	}
}

func createUser(r io.Reader) (u setting.UserType, e error) {
	e = json.NewDecoder(r).Decode(&u)
	return u, e
}
