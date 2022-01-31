package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func createToken(userId string) (string, error) {
	claims := jwt.MapClaims{"id": userId, "nbf": time.Now().In(time.UTC).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(settings.Secretkey))
}

func parseToken(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return settings.Secretkey, nil
	})

	if _, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// fmt.Println(claims["id"], claims["nbf"])
		return nil
	} else {
		return err
	}
}

func checkStartBody(r io.ReadCloser) (user, string, error) {
	body, e := ioutil.ReadAll(r)
	if e != nil {
		return user{}, "Unauthorized", e
	}
	b := user{}
	e = json.Unmarshal(body, &b)
	if e != nil {
		return user{}, "Unauthorized", e
	}

	if len(b.Id) == 0 || len(b.Password) == 0 {
		var m string = ""
		if len(b.Id) == 0 {
			m += `"idMessage":"input required."`
		}
		if len(b.Password) == 0 {
			if len(m) > 0 {
				m += ","
			}
			m += `"pwMessage":"input required."`
		}
		m = `{` + m + `}`
		return user{}, m, errors.New("Unauthorized")
	}

	for _, u := range settings.Users {
		if b.Id == u.Id && b.Password == u.Password {
			return u, "Unauthorized", nil
		}
	}

	return user{}, "Unauthorized", errors.New("Unauthorized")
}
