package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/multios12/auth-service/setting"
)

// トークンを作成する
func createToken(userId string) (string, error) {
	now := time.Now().In(time.UTC)
	claims := jwt.MapClaims{
		"id":  userId,
		"nbf": now.Unix(),
		"exp": now.AddDate(0, 0, 7).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(setting.Settings.Secretkey))
}

// リクエストのトークンから、ユーザ情報を取得
func parseTokenFromCookie(r *http.Request) (setting.UserType, error) {
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

		for _, u := range setting.Settings.Users {
			if u.Id == id {
				return u, nil
			}
		}

		return setting.UserType{}, fmt.Errorf("ID notfound")
	}

	return setting.UserType{}, fmt.Errorf("invalid token")
}

func createUser(r io.Reader) (u setting.UserType, e error) {
	e = json.NewDecoder(r).Decode(&u)
	return u, e
}
