package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func createToken(userId string) (string, error) {
	claims := jwt.MapClaims{"id": userId, "nbf": time.Now().In(time.UTC).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(settings.Secretkey))
}

func parseToken(tokenString string) (user, error) {
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

		return user{}, fmt.Errorf("id notfound")
	} else {
		return user{}, err
	}
}

func checkStartBody(r io.Reader) (user, string, error) {
	body, e := ioutil.ReadAll(r)
	var b user = user{}
	if e == nil {
		e = json.Unmarshal(body, &b)
	}

	var m []string
	if len(b.Id) == 0 {
		m = append(m, `"idMessage":"input required."`)
	}
	if len(b.Password) == 0 {
		m = append(m, `"pwMessage":"input required."`)
	}
	if len(m) > 0 {
		message := fmt.Sprintf("{%s}", strings.Join(m, ","))
		return user{}, message, errors.New("unauthorized")
	}

	for _, u := range settings.Users {
		if b.Id == u.Id && b.Password == u.Password {
			return u, "authorized", nil
		}
	}

	if e == nil {
		e = errors.New("unauthorized")
	}
	return user{}, "unauthorized", e
}
