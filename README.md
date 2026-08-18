# 認証サービス

nginx の [`auth_request`](https://nginx.org/en/docs/http/ngx_http_auth_request_module.html) と組み合わせて使用する、Rust 製のシンプルな認証サービスです。

バージョン 2.0.0 から、実装言語を Go から Rust に変更しました。

## できること

- ログイン画面の表示
- ID／パスワードによるログイン
- パスキーの登録とログイン
- ログアウト
- nginx の `auth_request` 用認証チェック

## インストール

[Releases](https://github.com/multios12/auth-service/releases) から、利用するバージョンの次のファイルをダウンロードします。

- `auth-service-vX.Y.Z-linux-x86_64-musl.tar.gz`
- `generate_password_hash-vX.Y.Z-linux-x86_64-musl.tar.gz`

ダウンロード後、アーカイブを展開します。

```bash
tar -xzf auth-service-v2.0.0-linux-x86_64-musl.tar.gz
tar -xzf generate_password_hash-v2.0.0-linux-x86_64-musl.tar.gz
chmod +x auth-service generate_password_hash
```

必要に応じて、同じReleaseに掲載されている `.sha256` ファイルでダウンロード内容を検証してください。

## 設定

プロジェクトのルート、または実行時のカレントディレクトリに `setting.json` を作成します。設定ファイルが存在しない場合、サービスは起動しません。

```json
{
  "mode": 1,
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

| 項目 | 説明 |
| --- | --- |
| `mode` | 認証モード。`1`、`2`、`3` のいずれかを指定します。 |
| `secretkey` | 認証トークンの署名に使用する32バイト以上の秘密鍵です。推測困難なランダム値を設定してください。条件を満たさない場合、サービスは起動しません。 |
| `users` | ログイン可能なユーザーの一覧です。 |
| `users[].id` | ログインIDです。 |
| `users[].password` | Argon2形式のパスワードハッシュです。 |
| `users[].permission` | アプリケーション側で利用する権限情報です。 |
| `users[].tokenVersion` | 発行済みトークンを無効化するための世代番号です。 |
| `users[].userHandle` | パスキーログインでユーザーを識別する値です。未登録時は空文字列にします。 |
| `users[].passkeys` | 登録済みパスキーの一覧です。未登録時は空配列にします。 |

### パスワードハッシュの生成

平文パスワードは `setting.json` に保存せず、付属ツールでArgon2ハッシュを生成します。

```bash
./generate_password_hash 'your-password'
```

標準出力に表示されたハッシュを、対象ユーザーの `password` に設定してください。シェルの履歴にパスワードが残る可能性があるため、実運用では実行環境の取り扱いに注意してください。

## 認証モード

| mode | 動作 |
| --- | --- |
| `1` | ID／パスワードでログインします。 |
| `2` | ID／パスワードでログイン後、パスキー登録画面へ遷移します。 |
| `3` | 登録済みのパスキーでログインします。 |

`mode=2` でパスキーを登録すると、`setting.json` の `userHandle` と `passkeys` が更新されます。更新は一時ファイルを利用して行われるため、サービスを実行するユーザーには `setting.json` が存在するディレクトリへの書き込み権限が必要です。

パスキーの RP ID とOriginは、リクエストの `Host` ヘッダーから生成されます。例えば `auth.example.com` ではRP IDが `example.com`、Originが `https://auth.example.com` になります。`localhost` の場合に限り、開発用のHTTP Originを使用できます。IPアドレスをHostとしたパスキー操作には対応していません。

## 起動

```bash
./auth-service
```

サービスは `0.0.0.0:3000` で待ち受けます。本番コンテナでは3000番をホストへ公開せず、信頼できるリバースプロキシと同じ内部ネットワークからのみ接続できるようにしてください。

## API

すべてのモードで利用できるAPI:

| メソッド | パス | 説明 |
| --- | --- | --- |
| `GET` | `/auth/api/auth` | 認証状態を確認します。認証済みの場合は `202`、未認証の場合は `401` を返します。 |
| `GET` | `/auth/api/logout` | 認証Cookieを削除し、ログイン画面へリダイレクトします。 |

モードごとに利用できる画面とAPI:

| mode | メソッド | パス | 説明 |
| --- | --- | --- | --- |
| `1`, `2` | `GET` | `/auth/login.html` | ID／パスワードのログイン画面を表示します。 |
| `1`, `2` | `POST` | `/auth/api/login` | ID／パスワードでログインします。 |
| `2` | `GET` | `/auth/passkey-register.html` | 認証済みユーザにパスキー登録画面を表示します。未認証時はログイン画面へリダイレクトします。 |
| `2` | `POST` | `/auth/api/register/options` | パスキー登録用のチャレンジを取得します。 |
| `2` | `POST` | `/auth/api/register/verify` | パスキー登録結果を検証し、`setting.json` に保存します。 |
| `3` | `GET` | `/auth/login.html` | パスキーログイン画面を表示します。 |
| `3` | `POST` | `/auth/api/login/options` | パスキーログイン用のチャレンジを取得します。 |
| `3` | `POST` | `/auth/api/login/verify` | パスキーログイン結果を検証してログインします。 |

mode `3` のログイン画面は `/auth/login.html` に統一しています。重複していた `/auth/passkey-login.html` は公開せず、アクセス時は `404 Not Found` となります。

## nginx 設定例

```nginx
location = /_auth {
    internal;
    proxy_pass http://127.0.0.1:3000/auth/api/auth;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header Cookie $http_cookie;
}

location /protected/ {
    auth_request /_auth;
    error_page 401 =302 /auth/login.html;
}

location /auth/ {
    proxy_pass http://127.0.0.1:3000;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

パスキーを利用する本番環境ではHTTPSを使用し、アプリケーションへ正しい `Host` ヘッダーを転送してください。  
移行前に既存の設定ファイルをバックアップし、上記の設定例に合わせて変換してください。

## 開発コンテナでのnginx連携テスト

`.devcontainer` から開発コンテナを起動すると、Rust開発用の `app` コンテナとリバースプロキシ用の `nginx` コンテナが同時に起動します。初回はテスト設定を用意し、認証サービスを開発コンテナ内で起動してください。

```bash
cp src/testdata/setting.json setting.json
cargo run
```

ブラウザで `http://localhost:8080/` を開き、`/private/` へ移動すると、nginx の `auth_request` を経由して未認証時にログイン画面が表示されます。ログイン後に再度 `/private/` を開くと、保護されたテストページを表示できます。

- `http://localhost:8080/`: nginx経由の公開テストページ
- `http://localhost:8080/private/`: nginxの認証対象ページ
- `http://localhost:8080/auth/login.html`: nginx経由のログイン画面
- `http://localhost:3000/auth/login.html`: Rustサービスへの直接アクセス

パスキーを試す場合は `setting.json` の `mode` を `2` にして登録し、その後 `3` に変更してサービスを再起動します。nginxは元の `Host` ヘッダーをポート込みで転送するため、`localhost:8080` のWebAuthn Originとして登録・認証できます。

## 参考

- [oauth2-proxy](https://github.com/oauth2-proxy/oauth2-proxy)
- [nginx auth_request module](https://nginx.org/en/docs/http/ngx_http_auth_request_module.html)
