use std::{
    collections::{HashMap, VecDeque},
    net::IpAddr,
    sync::{Arc, LazyLock, Mutex},
    time::{Duration, Instant},
};

use crate::{
    app_log,
    setting::{AppConfig, ModeEnum},
};
use argon2::{Argon2, PasswordHash, PasswordVerifier};
use axum::{
    Json,
    extract::State,
    http::{HeaderMap, StatusCode},
};
use axum_extra::extract::cookie::{Cookie, CookieJar};
use serde::{Deserialize, Serialize};

use crate::webauthn_common::create_token;

#[derive(Debug, Serialize, Deserialize)]
pub struct LoginRequest {
    id: String,
    password: String,
}

#[derive(Serialize)]
pub struct LoginResponse {
    next: &'static str,
}

const IP_LIMIT: usize = 20;
const IP_WINDOW: Duration = Duration::from_secs(5 * 60);
const LOGIN_LIMIT: usize = 5;
const LOGIN_WINDOW: Duration = Duration::from_secs(10 * 60);

#[derive(Default)]
struct LoginRateLimiter {
    by_ip: HashMap<IpAddr, VecDeque<Instant>>,
    by_login: HashMap<(IpAddr, String), VecDeque<Instant>>,
}

static LOGIN_RATE_LIMITER: LazyLock<Mutex<LoginRateLimiter>> =
    LazyLock::new(|| Mutex::new(LoginRateLimiter::default()));

/** APIハンドラ：ログイン */
pub async fn handler_login(
    State(config): State<Arc<AppConfig>>,
    jar: CookieJar,
    headers: HeaderMap,
    Json(body): Json<LoginRequest>,
) -> Result<(CookieJar, Json<LoginResponse>), StatusCode> {
    if config.users.is_empty() {
        app_log::warn_write("login_failed", "reason=no_configured_users");
        return Err(StatusCode::UNAUTHORIZED);
    }

    // 入力チェック
    if body.id.is_empty() || body.password.is_empty() {
        app_log::warn_write("login_failed", "reason=invalid_input");
        return Err(StatusCode::UNAUTHORIZED);
    }

    // 入力回数チェック
    //   ※WebサーバCaddy以外からこのコンテナへ直接接続できないことを前提にする。
    let client_ip = client_ip(&headers).ok_or_else(|| {
        app_log::warn_write("login_failed", "reason=invalid_forwarded_ip");
        StatusCode::BAD_REQUEST
    })?;
    if login_rate_limited(client_ip, &body.id) {
        app_log::warn_write("login_rate_limited", "reason=attempt_limit_exceeded");
        return Err(StatusCode::TOO_MANY_REQUESTS);
    }

    // ユーザ検索
    let Some(user) = config.users.iter().find(|user| user.id == body.id) else {
        record_login_failure(client_ip, &body.id);
        app_log::warn_write("login_failed", "reason=user_not_found");
        return Err(StatusCode::UNAUTHORIZED);
    };

    // パスワードの検証
    match verify_password(&body.password, &user.password) {
        Ok(true) => {}
        Ok(false) => {
            record_login_failure(client_ip, &body.id);
            app_log::warn_write("login_failed", "reason=password_mismatch");
            return Err(StatusCode::UNAUTHORIZED);
        }
        Err(error) => {
            record_login_failure(client_ip, &body.id);
            app_log::err_write(
                "login_failed",
                &format!("reason=invalid_password_hash error={error}"),
            );
            return Err(StatusCode::UNAUTHORIZED);
        }
    }

    clear_login_failures(client_ip, &body.id);

    let token = create_token(user, &config.secretkey)?;

    // トークンをクッキーに保存
    let cookie = Cookie::build(("token", token))
        .http_only(true)
        .secure(true)
        .path("/")
        .build();
    let jar = jar.add(cookie);
    app_log::info_write("login_succeeded", "method=password");

    Ok((
        jar,
        Json(LoginResponse {
            next: next_path(&config.mode),
        }),
    ))
}

/** 認証モードに応じたログイン後の遷移先を返す。 */
fn next_path(mode: &ModeEnum) -> &'static str {
    match mode {
        ModeEnum::PasskeyRegist => "/auth/passkey-register.html",
        _ => "/",
    }
}

/** パスワードを検証する */
fn verify_password(
    input_password: &str,
    stored_hash: &str,
) -> Result<bool, argon2::password_hash::Error> {
    let parsed_hash = PasswordHash::new(stored_hash)?;
    Ok(Argon2::default()
        .verify_password(input_password.as_bytes(), &parsed_hash)
        .is_ok())
}

/** Caddyが設定したX-Forwarded-ForからクライアントIPを取得する。 */
fn client_ip(headers: &HeaderMap) -> Option<IpAddr> {
    headers
        .get("x-forwarded-for")?
        .to_str()
        .ok()?
        .split(',')
        .next()?
        .trim()
        .parse()
        .ok()
}

/** ログイン回数チェック */
fn login_rate_limited(ip: IpAddr, id: &str) -> bool {
    let now = Instant::now();
    let mut limiter = LOGIN_RATE_LIMITER
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner());

    prune(&mut limiter.by_ip, now, IP_WINDOW);
    prune(&mut limiter.by_login, now, LOGIN_WINDOW);

    limiter.by_ip.get(&ip).is_some_and(|v| v.len() >= IP_LIMIT)
        || limiter
            .by_login
            .get(&(ip, id.to_owned()))
            .is_some_and(|v| v.len() >= LOGIN_LIMIT)
}

/** ログイン失敗回数の記録 */
fn record_login_failure(ip: IpAddr, id: &str) {
    let now = Instant::now();
    let mut limiter = LOGIN_RATE_LIMITER
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner());

    prune(&mut limiter.by_ip, now, IP_WINDOW);
    prune(&mut limiter.by_login, now, LOGIN_WINDOW);
    limiter.by_ip.entry(ip).or_default().push_back(now);
    limiter
        .by_login
        .entry((ip, id.to_owned()))
        .or_default()
        .push_back(now);
}

/** ログイン失敗回数のクリア */
fn clear_login_failures(ip: IpAddr, id: &str) {
    LOGIN_RATE_LIMITER
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner())
        .by_login
        .remove(&(ip, id.to_owned()));
}

fn prune<K: std::hash::Hash + Eq>(
    entries: &mut HashMap<K, VecDeque<Instant>>,
    now: Instant,
    window: Duration,
) {
    entries.retain(|_, attempts| {
        while attempts
            .front()
            .is_some_and(|time| now.duration_since(*time) >= window)
        {
            attempts.pop_front();
        }
        !attempts.is_empty()
    });
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::http::HeaderValue;

    #[test]
    fn extracts_first_forwarded_ip() {
        let mut headers = HeaderMap::new();
        headers.insert(
            "x-forwarded-for",
            HeaderValue::from_static("203.0.113.10, 172.18.0.2"),
        );

        assert_eq!(client_ip(&headers), Some("203.0.113.10".parse().unwrap()));
    }

    #[test]
    fn rejects_invalid_forwarded_ip() {
        let mut headers = HeaderMap::new();
        headers.insert("x-forwarded-for", HeaderValue::from_static("unknown"));

        assert_eq!(client_ip(&headers), None);
    }

    #[test]
    fn redirects_to_passkey_registration_in_registration_mode() {
        assert_eq!(
            next_path(&ModeEnum::PasskeyRegist),
            "/auth/passkey-register.html"
        );
    }

    #[test]
    fn redirects_to_root_in_id_login_mode() {
        assert_eq!(next_path(&ModeEnum::IdLogin), "/");
    }
}
