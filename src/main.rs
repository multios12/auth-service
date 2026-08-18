use std::sync::Arc;

use axum::{
    Router,
    routing::{get, post},
};
use tokio::net::TcpListener;

use crate::{
    handler_auth::handler_auth,
    handler_login::handler_login,
    handler_logout::handler_logout,
    handler_passkey_opt::{handler_login_options, handler_login_verify},
    handler_passkey_reg::{handler_reg_options, handler_reg_verify},
    setting::{AppConfig, ModeEnum, get_app_config},
};

pub mod app_log;
pub mod handler_auth;
pub mod handler_html;
pub mod handler_login;
pub mod handler_logout;
pub mod handler_passkey_opt;
pub mod handler_passkey_reg;
pub mod setting;
pub mod webauthn_common;

/* スタートアップポイント */
#[tokio::main]
async fn main() {
    let config: AppConfig = match get_app_config() {
        Ok(config) => config,
        Err(error) => {
            app_log::err_write(
                "settings_read_failed",
                &format!("operation=startup_load path=./setting.json error={error}"),
            );
            app_log::err_write(
                "service_start_failed",
                &format!("reason=settings_load_failed error={error}"),
            );
            return;
        }
    };

    let mode = format!("{:?}", config.mode);
    let user_count = config.users.len();
    let router = create_router(config);

    let listener: TcpListener = match TcpListener::bind("0.0.0.0:3000").await {
        Ok(listener) => listener,
        Err(error) => {
            app_log::err_write(
                "service_start_failed",
                &format!("reason=bind_failed error={error}"),
            );
            return;
        }
    };
    app_log::info_write(
        "service_started",
        &format!("mode={mode} user_count={user_count} listen=0.0.0.0:3000"),
    );
    if let Err(error) = axum::serve(listener, router).await {
        app_log::err_write(
            "service_start_failed",
            &format!("reason=serve_failed error={error}"),
        );
    }
}

/** ルータ作成 */
fn create_router(config: AppConfig) -> Router {
    let router: Router<Arc<AppConfig>> = Router::new()
        .route("/auth/api/logout", get(handler_logout))
        .route("/auth/api/auth", get(handler_auth));

    let router = match config.mode {
        ModeEnum::IdLogin => router
            .route("/auth/api/login", post(handler_login))
            .route("/auth/login.html", get(handler_html::handler)),
        ModeEnum::PasskeyRegist => router
            .route("/auth/api/login", post(handler_login))
            .route("/auth/api/register/options", post(handler_reg_options))
            .route("/auth/api/register/verify", post(handler_reg_verify))
            .route("/auth/login.html", get(handler_html::handler))
            .route(
                "/auth/passkey-register.html",
                get(handler_html::handler_passkey_reg_html),
            ),
        ModeEnum::PasskeyLogin => router
            .route("/auth/api/login/options", post(handler_login_options))
            .route("/auth/api/login/verify", post(handler_login_verify))
            .route(
                "/auth/login.html",
                get(handler_html::handler_passkey_login_html),
            ),
    };
    router.with_state(Arc::new(config))
}
