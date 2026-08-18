use std::{
    collections::HashMap,
    sync::{Arc, LazyLock, Mutex},
    time::{Duration, Instant, SystemTime, UNIX_EPOCH},
};

use axum::{
    Json,
    extract::State,
    http::{HeaderMap, StatusCode},
};
use axum_extra::extract::CookieJar;
use jsonwebtoken::{DecodingKey, Validation, decode};
use serde::{Deserialize, Serialize};
use uuid::Uuid;
use webauthn_rs::prelude::{
    CreationChallengeResponse, Passkey, PasskeyRegistration, RegisterPublicKeyCredential, Webauthn,
};

use crate::{
    app_log,
    setting::{AppConfig, PasskeyType},
    webauthn_common::{load_settings, update_settings, webauthn_from_headers},
};

const REGISTRATION_LIFETIME: Duration = Duration::from_secs(5 * 60);

struct PendingRegistration {
    webauthn: Webauthn,
    state: PasskeyRegistration,
    user_handle: Uuid,
    created_at: Instant,
}

static PENDING_REGISTRATIONS: LazyLock<Mutex<HashMap<String, PendingRegistration>>> =
    LazyLock::new(|| Mutex::new(HashMap::new()));

#[derive(Debug, Deserialize)]
struct Claims {
    id: String,
    ver: i32,
}

#[derive(Serialize)]
pub struct RegisterResponse {
    next: &'static str,
}

fn registration_failure(level: &str, reason: &str, status: StatusCode) -> StatusCode {
    match level {
        "ERROR" => app_log::err_write("passkey_registration_failed", &format!("reason={reason}")),
        "WARN" => app_log::warn_write("passkey_registration_failed", &format!("reason={reason}")),
        "INFO" => app_log::info_write("passkey_registration_failed", &format!("reason={reason}")),
        _ => app_log::err_write(
            "passkey_registration_failed",
            &format!("reason={reason} invalid_level={level}"),
        ),
    }
    status
}

/** APIハンドラ：パスキー登録 */
pub async fn handler_reg_options(
    State(config): State<Arc<AppConfig>>,
    jar: CookieJar,
    headers: HeaderMap,
) -> Result<Json<CreationChallengeResponse>, StatusCode> {
    let user_id = authenticated_user(&config, &jar)?;
    let webauthn = webauthn_from_headers(&headers)
        .map_err(|status| registration_failure("WARN", "invalid_host", status))?;

    // 保存済みデータはsetting.jsonの最新状態を参照する。
    let settings = load_settings()
        .map_err(|status| registration_failure("ERROR", "settings_read_failed", status))?;
    let user = settings
        .users
        .iter()
        .find(|user| user.id == user_id)
        .ok_or_else(|| registration_failure("WARN", "user_not_found", StatusCode::UNAUTHORIZED))?;
    let user_handle = if user.userHandle.is_empty() {
        Uuid::new_v4()
    } else {
        Uuid::parse_str(&user.userHandle).map_err(|error| {
            app_log::err_write(
                "passkey_registration_failed",
                &format!("reason=invalid_user_handle error={error}"),
            );
            StatusCode::INTERNAL_SERVER_ERROR
        })?
    };
    let exclude_credentials = user
        .passkeys
        .iter()
        .map(|stored| serde_json::from_str::<Passkey>(&stored.credentialData))
        .collect::<Result<Vec<_>, _>>()
        .map_err(|error| {
            app_log::err_write(
                "passkey_registration_failed",
                &format!("reason=invalid_stored_credential error={error}"),
            );
            StatusCode::INTERNAL_SERVER_ERROR
        })?
        .into_iter()
        .map(|passkey| passkey.cred_id().clone())
        .collect::<Vec<_>>();

    let (options, state) = webauthn
        .start_passkey_registration(
            user_handle,
            &user.id,
            &user.id,
            (!exclude_credentials.is_empty()).then_some(exclude_credentials),
        )
        .map_err(|error| {
            app_log::err_write(
                "passkey_registration_failed",
                &format!("reason=challenge_creation_failed error={error}"),
            );
            StatusCode::INTERNAL_SERVER_ERROR
        })?;

    let mut pending = PENDING_REGISTRATIONS
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner());
    pending.retain(|_, registration| registration.created_at.elapsed() < REGISTRATION_LIFETIME);
    pending.insert(
        user_id,
        PendingRegistration {
            webauthn,
            state,
            user_handle,
            created_at: Instant::now(),
        },
    );
    Ok(Json(options))
}

/** APIハンドラ：パスキー登録 検証 */
pub async fn handler_reg_verify(
    State(config): State<Arc<AppConfig>>,
    jar: CookieJar,
    Json(credential): Json<RegisterPublicKeyCredential>,
) -> Result<Json<RegisterResponse>, StatusCode> {
    let user_id = authenticated_user(&config, &jar)?;
    // 一度取り出した登録状態は、検証失敗時も再利用させない。
    let registration = PENDING_REGISTRATIONS
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner())
        .remove(&user_id)
        .ok_or_else(|| {
            registration_failure("WARN", "challenge_not_found", StatusCode::BAD_REQUEST)
        })?;
    if registration.created_at.elapsed() >= REGISTRATION_LIFETIME {
        app_log::warn_write("passkey_registration_failed", "reason=challenge_expired");
        return Err(StatusCode::BAD_REQUEST);
    }
    let passkey = registration
        .webauthn
        .finish_passkey_registration(&credential, &registration.state)
        .map_err(|error| {
            app_log::warn_write(
                "passkey_registration_failed",
                &format!("reason=verification_failed error={error}"),
            );
            StatusCode::BAD_REQUEST
        })?;
    save_passkey(&user_id, registration.user_handle, &passkey).inspect_err(|status| {
        app_log::err_write(
            "passkey_registration_failed",
            &format!("reason=save_failed status={status}"),
        );
    })?;
    app_log::info_write("passkey_registration_succeeded", "method=passkey");
    Ok(Json(RegisterResponse { next: "/" }))
}

fn authenticated_user(config: &AppConfig, jar: &CookieJar) -> Result<String, StatusCode> {
    let token = jar
        .get("token")
        .ok_or_else(|| registration_failure("WARN", "token_missing", StatusCode::UNAUTHORIZED))?
        .value();
    let token = decode::<Claims>(
        token,
        &DecodingKey::from_secret(config.secretkey.as_bytes()),
        &Validation::default(),
    )
    .map_err(|error| {
        app_log::warn_write(
            "passkey_registration_failed",
            &format!("reason=invalid_token error={error}"),
        );
        StatusCode::UNAUTHORIZED
    })?;
    let user = config
        .users
        .iter()
        .find(|user| user.id == token.claims.id && user.tokenVersion == token.claims.ver)
        .ok_or_else(|| registration_failure("WARN", "user_not_found", StatusCode::UNAUTHORIZED))?;
    Ok(user.id.clone())
}

fn save_passkey(user_id: &str, user_handle: Uuid, passkey: &Passkey) -> Result<(), StatusCode> {
    let credential_data =
        serde_json::to_string(passkey).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let credential_value: serde_json::Value =
        serde_json::from_str(&credential_data).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let credential_id = serde_json::to_value(passkey.cred_id())
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
        .as_str()
        .ok_or(StatusCode::INTERNAL_SERVER_ERROR)?
        .to_owned();

    let public_key = credential_value
        .pointer("/cred/cred")
        .cloned()
        .unwrap_or(serde_json::Value::Null)
        .to_string();
    let sign_count = credential_value
        .pointer("/cred/counter")
        .and_then(serde_json::Value::as_u64)
        .unwrap_or(0)
        .try_into()
        .unwrap_or(i32::MAX);
    let aaguid = credential_value
        .pointer("/cred/aaguid")
        .and_then(serde_json::Value::as_str)
        .unwrap_or_default()
        .to_owned();
    let created_at = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
        .as_secs()
        .to_string();
    update_settings(|settings| {
        if settings.users.iter().any(|user| {
            user.passkeys
                .iter()
                .any(|stored| stored.credentialID == credential_id)
        }) {
            app_log::warn_write("passkey_registration_failed", "reason=duplicate_credential");
            return Err(StatusCode::CONFLICT);
        }
        let user = settings
            .users
            .iter_mut()
            .find(|user| user.id == user_id)
            .ok_or_else(|| {
                registration_failure("WARN", "user_not_found", StatusCode::UNAUTHORIZED)
            })?;
        if user.userHandle.is_empty() {
            user.userHandle = user_handle.to_string();
        } else if user.userHandle != user_handle.to_string() {
            app_log::warn_write("passkey_registration_failed", "reason=user_handle_conflict");
            return Err(StatusCode::CONFLICT);
        }
        user.passkeys.push(PasskeyType {
            credentialID: credential_id.clone(),
            publicKey: public_key,
            signCount: sign_count,
            AAGUID: aaguid,
            name: credential_id,
            createdAt: created_at,
            credentialData: credential_data,
        });
        Ok(())
    })
}
