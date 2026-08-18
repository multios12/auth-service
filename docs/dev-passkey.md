# パスキー仕様書（Rust 版）

## 1. 概要

本書は Rust 版 `auth-service` 2.0.0 の現行実装に基づくパスキー仕様を定義する。WebAuthn 処理には `webauthn-rs 0.5.5` を使用する。

- 既存の ID／パスワード認証を維持する
- ID／パスワード認証後にパスキーを登録できるようにする
- 登録済みパスキーで、ユーザ ID を入力せずにログインできるようにする
- パスキー認証成功後も既存認証と同じ JWT Cookie を発行する

## 2. 認証モード

起動時に `setting.json` の `mode` を読み込み、公開する画面と API を切り替える。

| mode | Rust 列挙値 | 動作 |
| --- | --- | --- |
| `1` | `IdLogin` | ID／パスワードログイン |
| `2` | `PasskeyRegist` | ID／パスワードログイン後にパスキー登録 |
| `3` | `PasskeyLogin` | 登録済みパスキーによるログイン |

`mode` は必須であり、`1`、`2`、`3` 以外の値、欠落、型不正は設定読込エラーとなる。設定を読み込めない場合、サービスは起動しない。起動後のルーティングと通常の認証処理には起動時に読み込んだ設定を使用する。

## 3. 設定データ

```json
{
  "mode": 2,
  "secretkey": "32バイト以上のランダムな秘密鍵",
  "users": [
    {
      "id": "test",
      "password": "$argon2id$v=19$m=19456,t=2,p=1$...",
      "permission": "",
      "tokenVersion": 0,
      "userHandle": "",
      "passkeys": []
    }
  ]
}
```

フィールド名は大文字・小文字を区別し、全フィールドを必須とする。

### 3.1 ユーザ情報

| フィールド | 型 | 内容 |
| --- | --- | --- |
| `id` | string | ログイン ID。WebAuthn の user name／display name にも使用する |
| `password` | string | Argon2 形式のパスワードハッシュ |
| `permission` | string | 利用側アプリケーション向けの権限情報 |
| `tokenVersion` | integer | JWT 無効化用の世代番号 |
| `userHandle` | string | WebAuthn ユーザハンドル。未登録時は空文字列、登録時に UUID v4 を保存する |
| `passkeys` | array | 登録済みパスキー |

### 3.2 パスキー情報

| フィールド | 型 | 内容 |
| --- | --- | --- |
| `credentialID` | string | credential ID の base64url 表現 |
| `publicKey` | string | ライブラリ内部 credential の公開鍵部分を JSON 文字列化した値 |
| `signCount` | integer | 署名カウンタ。`i32` 上限を超える場合は `i32::MAX` |
| `AAGUID` | string | 認証器の AAGUID。取得できない場合は空文字列 |
| `name` | string | 表示名。現行実装では `credentialID` と同じ値 |
| `createdAt` | string | 登録時刻の Unix time（秒）を10進文字列で保存した値 |
| `credentialData` | string | `webauthn-rs` の `Passkey` 全体を JSON 文字列化した値 |

検証と署名カウンタ更新には `credentialData` を使用する。その他は参照・表示用の複製データである。`credentialData` はライブラリのシリアライズ形式に依存するため、ライブラリ更新時には保存済みデータの互換性確認が必要となる。

## 4. 画面とモード別ルーティング

ルートはモードごとに限定して登録される。別モード用の URL は `404 Not Found` となる。

| mode | メソッド | パス | 内容 |
| --- | --- | --- | --- |
| 1, 2 | `GET` | `/auth/login.html` | ID／パスワードログイン画面 |
| 1, 2 | `POST` | `/auth/api/login` | ID／パスワード認証 |
| 2 | `GET` | `/auth/passkey-register.html` | 認証済みユーザ向けパスキー登録画面。未認証時はログイン画面へリダイレクト |
| 2 | `POST` | `/auth/api/register/options` | 登録 options の発行 |
| 2 | `POST` | `/auth/api/register/verify` | 登録 credential の検証・保存 |
| 3 | `GET` | `/auth/login.html` | パスキーログイン画面 |
| 3 | `POST` | `/auth/api/login/options` | 認証 options の発行 |
| 3 | `POST` | `/auth/api/login/verify` | assertion の検証・ログイン |

mode 3 の画面URLは `/auth/login.html` に統一する。重複する `/auth/passkey-login.html` はルート登録せず、`404 Not Found` とする。

全モードで `GET /auth/api/auth`（成功時 `202 Accepted`、失敗時 `401 Unauthorized`）と `GET /auth/api/logout`（Cookie 削除後に `/auth/login.html` へリダイレクト）を利用できる。

## 5. ID／パスワード認証と画面遷移

`POST /auth/api/login` は `{"id":"test","password":"..."}` を受け取る。Argon2 ハッシュ検証成功時に JWT を `token` Cookie に設定し、次の JSON を返す。

```json
{"next":"/auth/passkey-register.html"}
```

`next` は `mode=2` では `/auth/passkey-register.html`、それ以外は `/` となる。

パスワードログインには、信頼するリバースプロキシが付与する `X-Forwarded-For` の先頭 IP を使ったメモリ内レート制限がある。

- IP 単位: 5 分間に失敗 20 回
- IP とログイン ID の組み合わせ: 10 分間に失敗 5 回
- 制限超過: `429 Too Many Requests`
- ヘッダ欠落または不正 IP: `400 Bad Request`

## 6. パスキー登録

### 6.1 `POST /auth/api/register/options`

`token` Cookie が必須である。JWT の署名、有効期限、ユーザ ID、`tokenVersion` を起動時設定に対して検証する。

1. `setting.json` を再読込し、JWT のユーザ ID に対応する最新ユーザを取得する。
2. `userHandle` が空なら UUID v4 を生成する。既存値は UUID として解釈する。
3. 既存の `credentialData` を `Passkey` に復元し、その ID を `excludeCredentials` に設定する。
4. `start_passkey_registration` で `CreationChallengeResponse` を生成する。
5. 登録状態をユーザ ID をキーにメモリへ保存し、options を JSON で返す。

同じユーザが再要求すると以前の未完了状態は上書きされる。有効期限は 5 分で、options 発行時に期限切れエントリを削除する。

| 状態 | HTTP |
| --- | --- |
| 発行成功 | `200 OK` |
| Cookie なし、JWT 不正、ユーザ不一致 | `401 Unauthorized` |
| Host 不正 | `400 Bad Request` |
| 設定読込、保存済み credential、options 生成の異常 | `500 Internal Server Error` |

### 6.2 ブラウザ処理

登録画面は `challenge`、`user.id`、`excludeCredentials[].id` を base64url から `ArrayBuffer` に変換して `navigator.credentials.create()` を呼ぶ。credential の `id`、`rawId`、`type`、`authenticatorAttachment`、`clientExtensionResults`、`attestationObject`、`clientDataJSON`、`transports` を JSON 化する。バイナリ値は padding なし base64url とする。

### 6.3 `POST /auth/api/register/verify`

本文は `webauthn-rs` の `RegisterPublicKeyCredential` JSON 形式とする。`token` Cookie で特定したユーザの保留状態を取り出した時点で削除するため、成功・失敗を問わずチャレンジは一度だけ使用できる。

検証成功後は以下を行う。

1. 全ユーザを対象に credential ID の重複を拒否する。
2. 空の `userHandle` に登録開始時の UUID を保存する。既存値と異なる場合は競合として拒否する。
3. `PasskeyType` を対象ユーザの `passkeys` に追加する。
4. `setting.json` を一時ファイルへ書き、flush／fsync 後に rename して置換する。
5. `{"next":"/"}` を返し、画面は `/` へ遷移する。

設定更新はプロセス内 mutex で直列化し、元ファイルのパーミッションを一時ファイルへ引き継ぐ。サービス実行ユーザには設定ディレクトリへの作成・rename 権限が必要である。

| 状態 | HTTP |
| --- | --- |
| 登録成功 | `200 OK` |
| チャレンジなし／期限切れ、credential 検証失敗 | `400 Bad Request` |
| Cookie なし、JWT 不正、ユーザ不一致 | `401 Unauthorized` |
| credential ID 重複、userHandle 競合 | `409 Conflict` |
| 設定保存等の異常 | `500 Internal Server Error` |

## 7. パスキーログイン

### 7.1 `POST /auth/api/login/options`

ユーザ ID と既存認証 Cookie は要求しない。`setting.json` を再読込し、全ユーザの正常な `credentialData` を認証候補として `start_passkey_authentication` に渡す。破損 credential はエラーログを記録して除外する。

現行実装は登録済み credential を options の認証候補に含め、credential ID によってユーザを逆引きする。user handle による逆引きは行わない。

認証状態は UUID v4 のセッション ID をキーにメモリへ保存し、次の Cookie を発行する。

| 属性 | 値 |
| --- | --- |
| 名前 | `passkey_auth_session` |
| `HttpOnly` | 有効 |
| `Secure` | 有効 |
| `SameSite` | `Strict` |
| `Path` | `/auth/api/login` |

有効期限は 5 分、プロセス全体の上限は 1,024 件である。options 発行時に期限切れ状態を削除し、上限到達時は `429` を返す。再起動すると保留状態は失われる。

| 状態 | HTTP |
| --- | --- |
| 発行成功 | `200 OK` |
| 有効な登録済みパスキーがない | `401 Unauthorized` |
| Host 不正 | `400 Bad Request` |
| 保留状態が 1,024 件に到達 | `429 Too Many Requests` |
| 設定読込、options 生成の異常 | `500 Internal Server Error` |

### 7.2 ブラウザ処理

ログイン画面は表示直後に処理を開始する。`challenge` と `allowCredentials[].id` を `ArrayBuffer` に変換して `navigator.credentials.get()` を呼ぶ。assertion の `id`、`rawId`、`type`、`clientExtensionResults`、`authenticatorData`、`clientDataJSON`、`signature`、存在する場合は `userHandle` を JSON 化する。バイナリ値は padding なし base64url とする。

### 7.3 `POST /auth/api/login/verify`

本文は `webauthn-rs` の `PublicKeyCredential` JSON 形式とする。`passkey_auth_session` Cookie の保留状態を取り出した時点で削除する。

1. assertion を保存済み状態に対して検証する。
2. credential ID と一致する `credentialData` を持つユーザを検索する。
3. `update_credential` で状態を更新する。
4. `signCount` と `credentialData` を更新し、`setting.json` をアトミックに置換する。
5. 対象ユーザの JWT を `token` Cookie に設定し、セッション Cookie を削除する。
6. `{"next":"/"}` を返し、画面は `/` へ遷移する。

| 状態 | HTTP |
| --- | --- |
| ログイン成功 | `200 OK` |
| セッション不正、チャレンジなし／期限切れ、assertion 不正、credential 不一致 | `401 Unauthorized` |
| 設定更新、JWT 生成等の異常 | `500 Internal Server Error` |

## 8. JWT と Cookie

JWT は HS256 で署名し、ユーザ ID (`id`)、発行時刻 (`nbf`)、7 日後の失効時刻 (`exp`)、`tokenVersion` (`ver`) を持つ。パスワードログインとパスキーログインは同じ Cookie を使う。

| 属性 | 値 |
| --- | --- |
| 名前 | `token` |
| `HttpOnly` | 有効 |
| `Secure` | 有効 |
| `Path` | `/` |
| `SameSite` | 明示指定なし |

ログアウト時は Path `/` の `token` Cookie を削除する。

## 9. RP ID と Origin

RP 設定は各 options リクエストの `Host` ヘッダーから生成する。forwarded host／proto は参照しないため、リバースプロキシは公開 URL の Host を転送する必要がある。

| Host | RP ID／RP name | Origin |
| --- | --- | --- |
| `auth.example.com` | `example.com` | `https://auth.example.com/` |
| `auth.example.co.jp:8443` | `example.co.jp` | `https://auth.example.co.jp:8443/` |
| `localhost:3000` | `localhost` | `http://localhost:3000/` |

- 空の Host、および `/`、`\\`、`@`、`#`、`?` を含む値を拒否する
- ホスト名を小文字化し、末尾の `.` を除去する
- `localhost` のみ HTTP を許可する
- IP アドレスは拒否する
- 通常ドメインは Public Suffix List（`psl` crate）から登録可能ドメインを求める
- Origin は `https://` と元の Host（ポートを含む）から生成する

本番環境では HTTPS を必須とする。登録時と認証時の Origin が一致しなければ検証に失敗する。

## 10. チャレンジ管理と並行実行

| 用途 | キー | 有効期限 | 上限 |
| --- | --- | --- | --- |
| 登録 | ユーザ ID | 5 分 | 明示上限なし。同一ユーザは後勝ち |
| 認証 | ランダム UUID | 5 分 | 1,024 件 |

保留状態はプロセスメモリだけに保存し、検証開始時に削除する。再起動をまたぐ処理や複数インスタンス間の共有には対応しない。複数インスタンス運用では同一インスタンスへルーティングを固定するか、共有ストアへの変更が必要となる。

## 11. ログ

主に以下のイベント名で記録する。credential、JWT、パスワードそのものは記録しない。

- `passkey_registration_succeeded` / `passkey_registration_failed`
- `passkey_auth_succeeded` / `passkey_auth_failed`
- `passkey_session_rate_limited`
- `settings_read_failed` / `settings_write_failed`

失敗ログには `reason=...` を付与する。

## 12. セキュリティ要件と運用上の制約

- パスキー登録は有効な `token` Cookie を持つユーザだけに許可する。
- challenge、Origin、RP ID、署名、credential 状態の検証は `webauthn-rs` に委譲する。
- credential ID は全ユーザで一意とする。
- 認証成功ごとに署名カウンタを永続化する。
- `setting.json` はサービス実行ユーザ以外の書込みを制限する。
- 設定更新は起動時の `AppConfig` へ自動反映されない。パスキー処理は必要箇所で再読込するが、通常の JWT 認証や登録 API の JWT ユーザ照合には起動時設定も使う。ユーザや `tokenVersion` の運用変更後はサービスを再起動する。
- 認証 options は登録済み全 credential を候補にする。登録数増加時はレスポンスサイズと認証器 UI への影響を評価する。

## 13. テスト観点

- `mode=1`、`2`、`3` で当該モードのルートだけが公開される
- `mode` 欠落、不正値、不足フィールドで起動しない
- `localhost`、public suffix、不正 Host、IP アドレスの RP 導出を検証する
- 未ログイン、期限切れ、不正 credential、チャレンジ再利用を拒否する
- 既存 credential が `excludeCredentials` に含まれる
- credential ID 重複と userHandle 競合を `409` で拒否する
- 破損した `credentialData` を除外し、正常な credential は継続利用できる
- 認証保留 1,024 件で `429` となる
- 成功時に署名カウンタ、`credentialData`、JWT Cookie が更新される
- 並行設定更新を直列化し、アトミック置換後もファイル権限を維持する

## 14. 対象外

- パスキーの一覧、名称変更、削除 UI／API
- attestation に基づく認証器の許可・拒否
- 共有チャレンジストアと複数インスタンス間の設定同期
- user handle によるユーザ逆引き
- IP アドレスを RP ID とする WebAuthn
