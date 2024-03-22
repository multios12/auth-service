package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/multios12/auth-service/setting"
)

// トークンを作成する
func createToken(userId string) (string, error) {
	claims := jwt.MapClaims{"id": userId, "nbf": time.Now().In(time.UTC).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(setting.Settings.Secretkey))
}

// リクエストのトークンから、ユーザ情報を取得
func parseTokenFromCookie(r *http.Request) (setting.UserType, error) {
	if cookie, e := r.Cookie("_auth-proxy"); e != nil {
		return setting.UserType{}, e
	} else if u, e := parseToken(cookie.Value); e != nil {
		return setting.UserType{}, e
	} else {
		return u, e
	}
}

// トークンからユーザ情報を取得
func parseToken(tokenString string) (setting.UserType, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(setting.Settings.Secretkey), nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		for _, u := range setting.Settings.Users {
			if u.Id == claims["id"] {
				return u, nil
			}
		}

		return setting.UserType{}, fmt.Errorf("ID notfound")
	} else {
		return setting.UserType{}, err
	}
}

func createUser(r io.Reader) (u setting.UserType, e error) {
	body, e := ioutil.ReadAll(r)
	if e == nil {
		e = json.Unmarshal(body, &u)
	}
	return u, e
}
