use std::time::{SystemTime, UNIX_EPOCH};

/** 標準エラーへ1イベント1行でログを出力する。 */
pub fn err_write(event: &str, detail: &str) {
    write("ERR ", event, detail);
}

/** 標準エラーへ1イベント1行でログを出力する。 */
pub fn warn_write(event: &str, detail: &str) {
    write("WARN", event, detail);
}

/** 標準エラーへ1イベント1行でログを出力する。 */
pub fn info_write(event: &str, detail: &str) {
    write("INFO", event, detail);
}

/** 標準エラーへ1イベント1行でログを出力する。 */
fn write(level: &str, event: &str, detail: &str) {
    let timestamp_ms = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_millis())
        .unwrap_or(0);
    let detail = detail.replace(['\r', '\n'], " ");
    eprintln!("timestamp_ms={timestamp_ms} level={level} event={event} {detail}");
}
