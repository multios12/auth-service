use std::{
    collections::HashMap,
    sync::{Arc, LazyLock, Mutex},
    time::{Duration, Instant},
};

use axum::{
    Json,
    extract::State,
    http::{HeaderMap, StatusCode},
};
use axum_extra::extract::{CookieJar, cookie::Cookie};
use serde::{Serialize, de::DeserializeOwned};
use uuid::Uuid;
use webauthn_rs::prelude::{
    Passkey, PasskeyAuthentication, PublicKeyCredential, RequestChallengeResponse, Webauthn,
};

use crate::{
    app_log,
    setting::AppConfig,
    webauthn_common::{create_token, load_settings, update_settings, webauthn_from_headers},
};

const AUTHENTICATION_LIFETIME: Duration = Duration::from_secs(5 * 60);
const MAX_PENDING_AUTHENTICATIONS: usize = 1024;
const SESSION_COOKIE: &str = "passkey_auth_session";

struct PendingAuthentication {
    webauthn: Webauthn,
    state: PasskeyAuthentication,
    created_at: Instant,
}

static PENDING_AUTHENTICATIONS: LazyLock<Mutex<HashMap<Uuid, PendingAuthentication>>> =
    LazyLock::new(|| Mutex::new(HashMap::new()));

#[derive(Serialize)]
pub struct LoginResponse {
    next: &'static str,
}

fn auth_failure(level: &str, reason: &str, status: StatusCode) -> StatusCode {
    match level {
        "ERROR" => app_log::err_write("passkey_auth_failed", &format!("reason={reason}")),
        "WARN" => app_log::warn_write("passkey_auth_failed", &format!("reason={reason}")),
        "INFO" => app_log::info_write("passkey_auth_failed", &format!("reason={reason}")),
        _ => app_log::err_write(
            "passkey_auth_failed",
            &format!("reason={reason} invalid_level={level}"),
        ),
    }
    status
}

/** APIハンドラ：パスキーログイン */
pub async fn handler_login_options(
    headers: HeaderMap,
    jar: CookieJar,
) -> Result<(CookieJar, Json<RequestChallengeResponse>), StatusCode> {
    let webauthn = webauthn_from_headers(&headers)
        .map_err(|status| auth_failure("WARN", "invalid_host", status))?;
    let settings =
        load_settings().map_err(|status| auth_failure("ERROR", "settings_read_failed", status))?;
    let passkeys = deserialize_valid_passkeys(&settings);
    if passkeys.is_empty() {
        app_log::warn_write("passkey_auth_failed", "reason=no_registered_passkey");
        return Err(StatusCode::UNAUTHORIZED);
    }
    let (options, state) = webauthn
        .start_passkey_authentication(&passkeys)
        .map_err(|error| {
            app_log::err_write(
                "passkey_auth_failed",
                &format!("reason=challenge_creation_failed error={error}"),
            );
            StatusCode::INTERNAL_SERVER_ERROR
        })?;

    let session_id = Uuid::new_v4();
    let mut pending = PENDING_AUTHENTICATIONS
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner());
    pending
        .retain(|_, authentication| authentication.created_at.elapsed() < AUTHENTICATION_LIFETIME);
    if pending.len() >= MAX_PENDING_AUTHENTICATIONS {
        app_log::warn_write(
            "passkey_session_rate_limited",
            "reason=pending_session_limit",
        );
        return Err(StatusCode::TOO_MANY_REQUESTS);
    }
    pending.insert(
        session_id,
        PendingAuthentication {
            webauthn,
            state,
            created_at: Instant::now(),
        },
    );
    drop(pending);

    let session_cookie = Cookie::build((SESSION_COOKIE, session_id.to_string()))
        .http_only(true)
        .secure(true)
        .same_site(axum_extra::extract::cookie::SameSite::Strict)
        .path("/auth/api/login")
        .build();
    Ok((jar.add(session_cookie), Json(options)))
}

/** 破損した登録情報を隔離し、正常に読み込めるパスキーだけを返す。 */
fn deserialize_valid_passkeys(config: &AppConfig) -> Vec<Passkey> {
    deserialize_valid_credentials(config)
}

fn deserialize_valid_credentials<T: DeserializeOwned>(config: &AppConfig) -> Vec<T> {
    config
        .users
        .iter()
        .flat_map(|user| {
            user.passkeys.iter().filter_map(move |stored| {
                match serde_json::from_str::<T>(&stored.credentialData) {
                    Ok(credential) => Some(credential),
                    Err(error) => {
                        app_log::err_write(
                            "passkey_auth_failed",
                            &format!("reason=invalid_stored_credential error={error}"),
                        );
                        None
                    }
                }
            })
        })
        .collect()
}

/** APIハンドラ：パスキーログイン 検証 */
pub async fn handler_login_verify(
    State(config): State<Arc<AppConfig>>,
    jar: CookieJar,
    Json(credential): Json<PublicKeyCredential>,
) -> Result<(CookieJar, Json<LoginResponse>), StatusCode> {
    let session_cookie = jar
        .get(SESSION_COOKIE)
        .ok_or_else(|| auth_failure("WARN", "session_cookie_missing", StatusCode::UNAUTHORIZED))?;
    let session_id = Uuid::parse_str(session_cookie.value())
        .map_err(|_| auth_failure("WARN", "invalid_session_id", StatusCode::UNAUTHORIZED))?;
    // チャレンジは成功・失敗にかかわらず一度だけ使用する。
    let pending = PENDING_AUTHENTICATIONS
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner())
        .remove(&session_id)
        .ok_or_else(|| auth_failure("WARN", "challenge_not_found", StatusCode::UNAUTHORIZED))?;
    if pending.created_at.elapsed() >= AUTHENTICATION_LIFETIME {
        app_log::warn_write("passkey_auth_failed", "reason=challenge_expired");
        return Err(StatusCode::UNAUTHORIZED);
    }

    let credential_id = credential.get_credential_id().to_vec();
    let result = pending
        .webauthn
        .finish_passkey_authentication(&credential, &pending.state)
        .map_err(|error| {
            app_log::warn_write(
                "passkey_auth_failed",
                &format!("reason=verification_failed error={error}"),
            );
            StatusCode::UNAUTHORIZED
        })?;

    let mut token = None;
    update_settings(|settings| {
        let user = settings
            .users
            .iter_mut()
            .find(|user| {
                user.passkeys.iter().any(|stored| {
                    serde_json::from_str::<Passkey>(&stored.credentialData)
                        .is_ok_and(|passkey| passkey.cred_id().as_ref() == credential_id.as_slice())
                })
            })
            .ok_or(StatusCode::UNAUTHORIZED)?;
        let stored = user
            .passkeys
            .iter_mut()
            .find(|stored| {
                serde_json::from_str::<Passkey>(&stored.credentialData)
                    .is_ok_and(|passkey| passkey.cred_id().as_ref() == credential_id.as_slice())
            })
            .ok_or(StatusCode::UNAUTHORIZED)?;
        let mut passkey = serde_json::from_str::<Passkey>(&stored.credentialData)
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
        passkey
            .update_credential(&result)
            .ok_or(StatusCode::UNAUTHORIZED)?;
        stored.signCount = result.counter().try_into().unwrap_or(i32::MAX);
        stored.credentialData =
            serde_json::to_string(&passkey).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
        token = Some(create_token(user, &config.secretkey)?);
        Ok(())
    })
    .inspect_err(|status| {
        app_log::err_write(
            "passkey_auth_failed",
            &format!("reason=credential_update_failed status={status}"),
        );
    })?;

    let auth_cookie = Cookie::build(("token", token.ok_or(StatusCode::INTERNAL_SERVER_ERROR)?))
        .http_only(true)
        .secure(true)
        .path("/")
        .build();
    let expired_session = Cookie::build(SESSION_COOKIE)
        .path("/auth/api/login")
        .build();
    app_log::info_write("passkey_auth_succeeded", "method=passkey");
    Ok((
        jar.remove(expired_session).add(auth_cookie),
        Json(LoginResponse { next: "/" }),
    ))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::setting::{ModeEnum, PasskeyType, UserType};
    use serde::Deserialize;

    #[derive(Debug, Deserialize, PartialEq)]
    struct TestCredential {
        id: u32,
    }

    fn passkey(credential_id: &str, credential_data: &str) -> PasskeyType {
        PasskeyType {
            credentialID: credential_id.to_owned(),
            publicKey: String::new(),
            signCount: 0,
            AAGUID: String::new(),
            name: String::new(),
            createdAt: String::new(),
            credentialData: credential_data.to_owned(),
        }
    }

    #[test]
    fn ignores_invalid_credential_data_and_keeps_valid_credentials() {
        let config = AppConfig {
            mode: ModeEnum::PasskeyLogin,
            secretkey: String::new(),
            users: vec![UserType {
                id: "test-user".to_owned(),
                password: String::new(),
                permission: String::new(),
                tokenVersion: 0,
                userHandle: String::new(),
                passkeys: vec![
                    passkey("valid", r#"{"id": 42}"#),
                    passkey("broken", "not-json"),
                ],
            }],
        };

        let credentials = deserialize_valid_credentials::<TestCredential>(&config);

        assert_eq!(credentials, vec![TestCredential { id: 42 }]);
    }
}
