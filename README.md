# 認証サービス

nginx の `auth_request` と組み合わせて使うシンプルな認証サービスです。

## できること

- ログイン画面の表示
- ログイン / ログアウト
- 認証チェック
- ユーザ情報の取得
- パスワード変更
- パスキー登録
- パスキーログイン

## 起動方法

### 開発環境

必要なもの:

- VS Code
- Docker Desktop
- VS Code の `Remote - Containers` または `Dev Containers`

VS Code で `Ctrl+Shift+P` を押し、`Reopen in Container` を選ぶと開発環境を起動できます。

### サーバ起動

`server` ディレクトリで起動します。

```bash
go run . -port :3000 -filename ./setting.json
```

- `-port`: 待ち受けポート
- `-filename`: 設定ファイルのパス

## 設定ファイル

初回起動時は、指定した設定ファイルがなければ自動生成されます。

設定例:

```json
{"mode":1,"Secretkey":"xxxxxxxxxxxxxxxxxxxx","Users":[{"Id":"test","Password":"test","Permission":""}]}
```

- `mode`: 認証モード。未指定または `2` / `3` 以外の場合は `1` として扱います
- `Secretkey`: トークン署名用の秘密鍵
- `Users`: ログイン可能なユーザ一覧

### 認証モード

| mode | 動作 |
| --- | --- |
| `1` | ID / パスワードでログインします |
| `2` | ID / パスワードでログイン後、パスキー登録画面に遷移します |
| `3` | パスキーでログインします |

`mode=2` では `/auth/api/passkey/register/*` の登録 API だけが有効になります。
`mode=3` では `/auth/api/passkey/login/*` のログイン API だけが有効になります。
`mode=3` では ID / パスワードログイン用の `/auth/api/login` は無効になります。

パスキー登録後は、ユーザ情報に `UserHandle` と `Passkeys` が保存されます。
`UserHandle` は ID 入力なしのパスキーログインでユーザを特定するための値です。

パスキーの RP ID と表示名は設定ファイルには持たず、リクエストの `Host` から導出します。
例えば `auth.example.com` でアクセスした場合、RP ID と表示名は `example.com` になります。
`localhost` では開発用に `http://localhost:<port>` の Origin を許可します。

## API

- `GET /auth/login.html` - ログインページを表示
- `GET /auth/setting.html` - 設定ページを表示
- `POST /auth/api/login` - ログイン処理
- `GET /auth/api/logout` - ログアウト処理
- `GET /auth/api/auth` - nginx の `auth_request` 用認証チェック。`202` または `401` を返す
- `GET /auth/api/info` - ユーザ情報を取得
- `POST /auth/api/info` - パスワードを変更
- `POST /auth/api/passkey/register/options` - パスキー登録用のチャレンジを取得
- `POST /auth/api/passkey/register/verify` - パスキー登録結果を検証して保存
- `POST /auth/api/passkey/login/options` - パスキーログイン用のチャレンジを取得
- `POST /auth/api/passkey/login/verify` - パスキーログイン結果を検証してログイン

パスキー登録 API は `mode=2` のときだけ有効です。
パスキーログイン API は `mode=3` のときだけ有効です。

## ビルド確認

```bash
goreleaser check
```

## 参考

- https://github.com/oauth2-proxy/oauth2-proxy
- https://qiita.com/convto/items/2822d029349cb1b4df93
- https://qiita.com/OmeletteCurry19/items/f24ee02a942d8f6931a5
