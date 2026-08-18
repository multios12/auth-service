use std::sync::Arc;

use axum::{extract::State, response::Redirect};
use axum_extra::extract::{CookieJar, cookie::Cookie};

use crate::setting::{AppConfig, ModeEnum};

/** APIハンドラ：ログアウト */
pub async fn handler_logout(
    State(config): State<Arc<AppConfig>>,
    jar: CookieJar,
) -> (CookieJar, Redirect) {
    // クッキーのクリア
    let cookie = Cookie::build("token").path("/").removal().build();
    let jar = jar.add(cookie);

    if config.mode == ModeEnum::PasskeyLogin {
        return (jar, Redirect::to("/auth/login.html"));
    }
    (jar, Redirect::to("/auth/login.html"))
}
