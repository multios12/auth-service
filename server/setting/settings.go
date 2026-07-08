package setting

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// 設定ファイルを読み込む
func Read(filename string) error {
	settingsFile = filename

	_, e := os.Stat(filename)
	if e != nil {
		Settings = SettingsType{Secretkey: createRandomString(20), Users: []UserType{}}
		fmt.Printf("Load: setting file[%s]\n", settingsFile)
		return Write()
	}

	f, e := os.ReadFile(filename)
	if e == nil {
		e = json.Unmarshal(f, &Settings)
		if e == nil && len(Settings.Secretkey) == 0 {
			Settings.Secretkey = createRandomString(20)
			return Write()
		}
	}
	return e
}

// 有効な認証モードを返す
func Mode() int {
	if Settings.Mode == 2 || Settings.Mode == 3 {
		return Settings.Mode
	}
	return 1
}

// 設定ファイルを書き込む
func Write() error {
	b, _ := json.Marshal(Settings)
	return os.WriteFile(settingsFile, b, 0o600)
}

// 指定ユーザのパスワードを更新する
func UpdatePassword(id string, password string) error {
	for i := range Settings.Users {
		if Settings.Users[i].Id == id {
			oldPassword := Settings.Users[i].Password
			oldTokenVersion := Settings.Users[i].TokenVersion
			Settings.Users[i].Password = password
			Settings.Users[i].TokenVersion++
			if e := Write(); e != nil {
				Settings.Users[i].Password = oldPassword
				Settings.Users[i].TokenVersion = oldTokenVersion
				return e
			}
			return nil
		}
	}
	return fmt.Errorf("user notfound")
}

// 指定ユーザにWebAuthnユーザハンドルがなければ作成する
func EnsureUserHandle(id string) (string, error) {
	for i := range Settings.Users {
		if Settings.Users[i].Id == id {
			if Settings.Users[i].UserHandle != "" {
				return Settings.Users[i].UserHandle, nil
			}

			oldUserHandle := Settings.Users[i].UserHandle
			Settings.Users[i].UserHandle = createRandomString(43)
			if e := Write(); e != nil {
				Settings.Users[i].UserHandle = oldUserHandle
				return "", e
			}
			return Settings.Users[i].UserHandle, nil
		}
	}
	return "", fmt.Errorf("user notfound")
}

// 指定ユーザにパスキー情報を追加する
func AddPasskey(id string, passkey PasskeyType) error {
	for i := range Settings.Users {
		if Settings.Users[i].Id == id {
			oldPasskeys := append([]PasskeyType(nil), Settings.Users[i].Passkeys...)
			if passkey.CreatedAt == "" {
				passkey.CreatedAt = time.Now().In(time.UTC).Format(time.RFC3339)
			}
			for j := range Settings.Users[i].Passkeys {
				if Settings.Users[i].Passkeys[j].CredentialID == passkey.CredentialID {
					Settings.Users[i].Passkeys[j] = passkey
					if e := Write(); e != nil {
						Settings.Users[i].Passkeys = oldPasskeys
						return e
					}
					return nil
				}
			}
			Settings.Users[i].Passkeys = append(Settings.Users[i].Passkeys, passkey)
			if e := Write(); e != nil {
				Settings.Users[i].Passkeys = oldPasskeys
				return e
			}
			return nil
		}
	}
	return fmt.Errorf("user notfound")
}

// 指定した認証情報IDのパスキー情報を更新する
func UpdatePasskey(credentialID string, passkey PasskeyType) error {
	for i := range Settings.Users {
		for j := range Settings.Users[i].Passkeys {
			if Settings.Users[i].Passkeys[j].CredentialID == credentialID {
				oldPasskey := Settings.Users[i].Passkeys[j]
				Settings.Users[i].Passkeys[j] = passkey
				if e := Write(); e != nil {
					Settings.Users[i].Passkeys[j] = oldPasskey
					return e
				}
				return nil
			}
		}
	}
	return fmt.Errorf("passkey notfound")
}

// 指定したユーザIDのユーザ情報を返す
func FindUserByID(id string) (UserType, bool) {
	for _, u := range Settings.Users {
		if u.Id == id {
			return u, true
		}
	}
	return UserType{}, false
}

// 指定したWebAuthnユーザハンドルのユーザ情報を返す
func FindUserByHandle(userHandle string) (UserType, bool) {
	for _, u := range Settings.Users {
		if u.UserHandle == userHandle {
			return u, true
		}
	}
	return UserType{}, false
}

// 指定した認証情報IDのユーザ情報を返す
func FindUserByCredentialID(credentialID string) (UserType, bool) {
	for _, u := range Settings.Users {
		for _, p := range u.Passkeys {
			if p.CredentialID == credentialID {
				return u, true
			}
		}
	}
	return UserType{}, false
}

// 指定長のランダム文字列を生成する
func createRandomString(digit uint32) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, digit)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}

	var result string
	for _, v := range b {
		result += string(letters[int(v)%len(letters)])
	}
	return result
}
