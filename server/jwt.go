package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func createToken(userId string) (string, error) {
	claims := jwt.MapClaims{"id": userId, "nbf": time.Now().In(time.UTC).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(settings.Secretkey))
}

func parseTokenFromCookie(r *http.Request) (userType, error) {
	if cookie, e := r.Cookie("_auth-proxy"); e != nil {
		return userType{}, e
	} else if u, e := parseToken(cookie.Value); e != nil {
		return userType{}, e
	} else {
		return u, e
	}
}

func parseToken(tokenString string) (userType, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(settings.Secretkey), nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		for _, u := range settings.Users {
			if u.Id == claims["id"] {
				return u, nil
			}
		}

		return userType{}, fmt.Errorf("ID notfound")
	} else {
		return userType{}, err
	}
}

func createUser(r io.Reader) (u userType, e error) {
	body, e := ioutil.ReadAll(r)
	if e == nil {
		e = json.Unmarshal(body, &u)
	}
	return u, e
}

func (u userType) Check() error {
	if len(u.Id) == 0 {
		return fmt.Errorf(`ID input required.`)
	} else if len(u.Password) == 0 {
		return fmt.Errorf(`PASSWORD input required.`)
	}

	return nil
}

func (u userType) CheckUser() error {
	for _, t := range settings.Users {
		if t.Id == u.Id && t.Password == u.Password {
			return nil
		}
	}

	return fmt.Errorf("ID / PASSWORD do not match.")
}
