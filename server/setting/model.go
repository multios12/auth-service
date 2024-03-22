package setting

import "fmt"

var settingsFile string   // 設定ファイルパス
var Settings SettingsType //設定

// ユーザ
type UserType struct {
	Id         string // ユーザID
	Password   string // パスワード
	Permission string // 権限
}

// 設定
type SettingsType struct {
	Secretkey string     // 秘密鍵
	Users     []UserType // ユーザ情報
}
type ChangeType struct {
	OldPassword string // 以前のパスワード
	NewPassword string // 新しいパスワード
}

func (u UserType) Check() error {
	if len(u.Id) == 0 {
		return fmt.Errorf(`ID input required.\n`)
	} else if len(u.Password) == 0 {
		return fmt.Errorf(`PASSWORD input required.\n`)
	}

	return nil
}

func (u UserType) CheckUser() error {
	for _, t := range Settings.Users {
		if t.Id == u.Id && t.Password == u.Password {
			return nil
		}
	}

	return fmt.Errorf(`ID / PASSWORD do not match.\n`)
}
