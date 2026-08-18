use std::{
    fs::{self, File, OpenOptions},
    io::{BufReader, BufWriter, Write},
    sync::{LazyLock, Mutex},
    time::{SystemTime, UNIX_EPOCH},
};

use axum::http::{HeaderMap, StatusCode, header::HOST};
use jsonwebtoken::{EncodingKey, Header, encode};
use serde::Serialize;
use uuid::Uuid;
use webauthn_rs::prelude::{Url, Webauthn, WebauthnBuilder};

use crate::{
    app_log,
    setting::{AppConfig, UserType},
};

const SETTINGS_PATH: &str = "./setting.json";
static SETTINGS_WRITE_LOCK: LazyLock<Mutex<()>> = LazyLock::new(|| Mutex::new(()));

#[derive(Serialize)]
struct Claims {
    id: String,
    nbf: usize,
    exp: usize,
    ver: i32,
}

/** HTTPヘッダから、webAuthn情報(RPID, RPName, RFDisplayName)を返す */
pub fn webauthn_from_headers(headers: &HeaderMap) -> Result<Webauthn, StatusCode> {
    let host = headers
        .get(HOST)
        .and_then(|value| value.to_str().ok())
        .ok_or(StatusCode::BAD_REQUEST)?;
    let (rp_id, origin) = relying_party(host)?;
    WebauthnBuilder::new(&rp_id, &origin)
        .map_err(|_| StatusCode::BAD_REQUEST)?
        .rp_name(&rp_id)
        .build()
        .map_err(|_| StatusCode::BAD_REQUEST)
}

/** 指定されたホストからRFIDとorignを生成する */
pub fn relying_party(host_header: &str) -> Result<(String, Url), StatusCode> {
    if host_header.is_empty()
        || host_header
            .chars()
            .any(|c| matches!(c, '/' | '\\' | '@' | '#' | '?'))
    {
        return Err(StatusCode::BAD_REQUEST);
    }
    let parsed =
        Url::parse(&format!("http://{host_header}")).map_err(|_| StatusCode::BAD_REQUEST)?;
    let hostname = parsed
        .host_str()
        .ok_or(StatusCode::BAD_REQUEST)?
        .trim_end_matches('.')
        .to_ascii_lowercase();
    if hostname == "localhost" {
        let origin =
            Url::parse(&format!("http://{host_header}")).map_err(|_| StatusCode::BAD_REQUEST)?;
        return Ok((hostname, origin));
    }
    if hostname.parse::<std::net::IpAddr>().is_ok() {
        return Err(StatusCode::BAD_REQUEST);
    }
    let rp_id = psl::domain_str(&hostname)
        .ok_or(StatusCode::BAD_REQUEST)?
        .trim_end_matches('.')
        .to_owned();
    let origin =
        Url::parse(&format!("https://{host_header}")).map_err(|_| StatusCode::BAD_REQUEST)?;
    Ok((rp_id, origin))
}

/** setting.jsonを読み込む */
pub fn load_settings() -> Result<AppConfig, StatusCode> {
    let file = File::open(SETTINGS_PATH).map_err(|error| {
        app_log::err_write(
            "settings_read_failed",
            &format!("operation=open path={SETTINGS_PATH} error={error}"),
        );
        StatusCode::INTERNAL_SERVER_ERROR
    })?;
    let settings: AppConfig = serde_json::from_reader(BufReader::new(file)).map_err(|error| {
        app_log::err_write(
            "settings_read_failed",
            &format!("operation=parse path={SETTINGS_PATH} error={error}"),
        );
        StatusCode::INTERNAL_SERVER_ERROR
    })?;
    settings.validate().map_err(|error| {
        app_log::err_write(
            "settings_read_failed",
            &format!("operation=validate path={SETTINGS_PATH} error={error}"),
        );
        StatusCode::INTERNAL_SERVER_ERROR
    })?;
    Ok(settings)
}

/** setting.jsonを更新する */
pub fn update_settings(
    update: impl FnOnce(&mut AppConfig) -> Result<(), StatusCode>,
) -> Result<(), StatusCode> {
    let _guard = SETTINGS_WRITE_LOCK
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner());
    let mut settings = load_settings()?;
    update(&mut settings)?;
    write_settings_atomically(&settings)
}

/** ユーザ情報とシークレットから、トークンを生成する */
pub fn create_token(user: &UserType, secret: &str) -> Result<String, StatusCode> {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|error| {
            app_log::err_write(
                "token_creation_failed",
                &format!("operation=clock error={error}"),
            );
            StatusCode::INTERNAL_SERVER_ERROR
        })?
        .as_secs() as usize;
    encode(
        &Header::default(),
        &Claims {
            id: user.id.clone(),
            nbf: now,
            exp: now + 60 * 60 * 24 * 7,
            ver: user.tokenVersion,
        },
        &EncodingKey::from_secret(secret.as_bytes()),
    )
    .map_err(|error| {
        app_log::err_write(
            "token_creation_failed",
            &format!("operation=encode error={error}"),
        );
        StatusCode::INTERNAL_SERVER_ERROR
    })
}

/** setting.jsonを書き込む */
fn write_settings_atomically(settings: &AppConfig) -> Result<(), StatusCode> {
    let temporary_path = format!("{SETTINGS_PATH}.{}.tmp", Uuid::new_v4());
    let result = (|| {
        let permissions = fs::metadata(SETTINGS_PATH)?.permissions();
        let file = OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(&temporary_path)?;
        file.set_permissions(permissions)?;
        let mut writer = BufWriter::new(file);
        serde_json::to_writer_pretty(&mut writer, settings)?;
        writer.write_all(b"\n")?;
        writer.flush()?;
        writer.get_ref().sync_all()?;
        fs::rename(&temporary_path, SETTINGS_PATH)?;
        Ok::<_, Box<dyn std::error::Error>>(())
    })();
    if let Err(error) = result {
        app_log::err_write(
            "settings_write_failed",
            &format!("path={SETTINGS_PATH} error={error}"),
        );
        let _ = fs::remove_file(&temporary_path);
        return Err(StatusCode::INTERNAL_SERVER_ERROR);
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn derives_registrable_domain_and_https_origin() {
        let (rp_id, origin) = relying_party("auth.example.co.jp:8443").unwrap();
        assert_eq!(rp_id, "example.co.jp");
        assert_eq!(origin.as_str(), "https://auth.example.co.jp:8443/");
    }

    #[test]
    fn permits_http_only_for_localhost() {
        let (rp_id, origin) = relying_party("localhost:3000").unwrap();
        assert_eq!(rp_id, "localhost");
        assert_eq!(origin.as_str(), "http://localhost:3000/");
    }

    #[test]
    fn rejects_ip_address_and_invalid_authority() {
        assert!(relying_party("127.0.0.1:3000").is_err());
        assert!(relying_party("example.com/path").is_err());
        assert!(relying_party("user@example.com").is_err());
    }
}
