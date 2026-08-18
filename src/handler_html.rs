use std::sync::Arc;

use axum::{
    extract::State,
    response::{Html, Redirect},
};
use axum_extra::extract::CookieJar;

use crate::{handler_auth::is_authenticated, setting::AppConfig};

pub async fn handler() -> Html<&'static str> {
    Html(include_str!("./static/login.html"))
}

pub async fn handler_passkey_login_html() -> Html<&'static str> {
    Html(include_str!("./static/passkey-login.html"))
}

pub async fn handler_passkey_reg_html(
    State(config): State<Arc<AppConfig>>,
    jar: CookieJar,
) -> Result<Html<&'static str>, Redirect> {
    if !is_authenticated(&config, &jar) {
        return Err(Redirect::to("/auth/login.html"));
    }

    Ok(Html(include_str!("./static/passkey-register.html")))
}
