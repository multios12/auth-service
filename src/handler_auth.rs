use std::sync::Arc;

use axum::{extract::State, http::StatusCode};
use axum_extra::extract::CookieJar;
use jsonwebtoken::{Algorithm, DecodingKey, Validation, decode};
use serde::{Deserialize, Serialize};

use crate::setting::AppConfig;

#[derive(Debug, Serialize, Deserialize)]
struct Claims {
    id: String,
    nbf: usize,
    exp: usize,
    ver: i32,
}

/** APIハンドラ：認証 */
pub async fn handler_auth(
    State(config): State<Arc<AppConfig>>,
    jar: CookieJar,
) -> Result<String, StatusCode> {
    if !is_authenticated(&config, &jar) {
        return Err(StatusCode::UNAUTHORIZED);
    }

    Err(StatusCode::ACCEPTED)
}

/** CookieのJWTが現在のユーザ情報に対して有効か確認する。 */
pub fn is_authenticated(config: &AppConfig, jar: &CookieJar) -> bool {
    let Some(token) = jar.get("token") else {
        return false;
    };
    let decoded = decode::<Claims>(
        token.value(),
        &DecodingKey::from_secret(config.secretkey.as_bytes()),
        &Validation::new(Algorithm::HS256),
    );
    let Ok(decoded) = decoded else {
        return false;
    };

    config
        .users
        .iter()
        .any(|user| decoded.claims.id == user.id && decoded.claims.ver == user.tokenVersion)
}
