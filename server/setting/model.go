package setting

import "fmt"

var settingsFile string   // 設定ファイルパス
var Settings SettingsType //設定

// ユーザ情報
type UserType struct {
	Id           string `json:"Id"`         // ユーザID
	Password     string `json:"Password"`   // パスワード
	Permission   string `json:"Permission"` // 権限
	TokenVersion int    `json:",omitempty"` // JWT 無効化用の世代番号
}

// 設定情報
type SettingsType struct {
	Secretkey string     `json:"Secretkey"` // 秘密鍵
	Users     []UserType `json:"Users"`     // ユーザ情報
}

// パスワード変更内容
type ChangeType struct {
	OldPassword string `json:"OldPassword"` // 以前のパスワード
	NewPassword string `json:"NewPassword"` // 新しいパスワード
}

// 入力値を確認する
func (u UserType) Check() error {
	if len(u.Id) == 0 {
		return fmt.Errorf(`ID input required.\n`)
	} else if len(u.Password) == 0 {
		return fmt.Errorf(`PASSWORD input required.\n`)
	}

	return nil
}

// 登録済みユーザと一致するか確認する
func (u UserType) CheckUser() error {
	for _, t := range Settings.Users {
		if t.Id == u.Id && t.Password == u.Password {
			return nil
		}
	}

	return fmt.Errorf(`ID / PASSWORD do not match.\n`)
}
