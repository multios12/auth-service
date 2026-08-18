use serde::{Deserialize, Serialize};
use serde_repr::{Deserialize_repr, Serialize_repr};

use std::{fs::File, io::BufReader};

pub const MIN_SECRET_KEY_LENGTH: usize = 32;

// let settingsFile string   // 設定ファイルパス
// let Settings SettingsType // 設定

/** 認証モード */
#[derive(Serialize_repr, Deserialize_repr, Debug)]
#[allow(non_snake_case)]
#[repr(u8)]
#[derive(PartialEq)]
pub enum ModeEnum {
    IdLogin = 1,       // ID認証モード
    PasskeyRegist = 2, // パスキー登録モード
    PasskeyLogin = 3,  // パスキー認証モード
}

/** 設定情報 */
#[derive(Serialize, Deserialize, Debug)]
pub struct AppConfig {
    /** 認証モード */
    pub mode: ModeEnum,
    /** 秘密鍵 */
    pub secretkey: String,
    /**  ユーザ情報 */
    pub users: Vec<UserType>,
}

/** ユーザ情報 */
#[derive(Serialize, Deserialize, Debug)]
#[allow(non_snake_case)]
pub struct UserType {
    pub id: String,                 // ユーザID
    pub password: String,           // パスワード
    pub permission: String,         // 権限
    pub tokenVersion: i32,          // JWT 無効化用の世代番号
    pub userHandle: String,         // WebAuthnユーザハンドル
    pub passkeys: Vec<PasskeyType>, // 登録済みパスキー
}

/** パスキー情報 */
#[derive(Serialize, Deserialize, Debug)]
#[allow(non_snake_case)]
pub struct PasskeyType {
    /** 認証器が発行した認証情報ID */
    pub credentialID: String,
    /** 公開鍵 */
    pub publicKey: String,
    /** 署名カウンタ */
    pub signCount: i32,
    /** 認証器のAAGUID */
    pub AAGUID: String,
    /** 表示名 */
    pub name: String,
    /** 作成日時 */
    pub createdAt: String,
    /** WebAuthnライブラリ用の認証情報JSON */
    pub credentialData: String,
}

pub fn get_app_config() -> Result<AppConfig, Box<dyn std::error::Error>> {
    let file = File::open("./setting.json")?;
    let rdr = BufReader::new(file);
    let setting: AppConfig = serde_json::from_reader(rdr)?;
    setting.validate()?;

    Ok(setting)
}

impl AppConfig {
    /** JWT署名鍵を含む設定値を検証する。 */
    pub fn validate(&self) -> Result<(), Box<dyn std::error::Error>> {
        if self.secretkey.len() < MIN_SECRET_KEY_LENGTH {
            return Err(format!("secretkey must be at least {MIN_SECRET_KEY_LENGTH} bytes").into());
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn config_with_secret(secretkey: &str) -> AppConfig {
        AppConfig {
            mode: ModeEnum::IdLogin,
            secretkey: secretkey.to_owned(),
            users: Vec::new(),
        }
    }

    #[test]
    fn rejects_secret_key_shorter_than_32_bytes() {
        assert!(config_with_secret(&"a".repeat(31)).validate().is_err());
    }

    #[test]
    fn accepts_secret_key_of_at_least_32_bytes() {
        assert!(config_with_secret(&"a".repeat(32)).validate().is_ok());
    }
}
