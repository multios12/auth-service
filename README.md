# 認証サービス

nginx の `auth_request` と組み合わせて使うシンプルな認証サービスです。

## できること

- ログイン画面の表示
- ログイン / ログアウト
- 認証チェック
- ユーザ情報の取得
- パスワード変更

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
{"Secretkey":"xxxxxxxxxxxxxxxxxxxx","Users":[{"Id":"test","Password":"test","Permission":""}]}
```

- `Secretkey`: トークン署名用の秘密鍵
- `Users`: ログイン可能なユーザ一覧

## API

- `GET /auth/login.html` - ログインページを表示
- `GET /auth/setting.html` - 設定ページを表示
- `POST /auth/api/login` - ログイン処理
- `GET /auth/api/logout` - ログアウト処理
- `GET /auth/api/auth` - nginx の `auth_request` 用認証チェック。`202` または `401` を返す
- `GET /auth/api/info` - ユーザ情報を取得
- `POST /auth/api/info` - パスワードを変更

## ビルド確認

```bash
goreleaser check
```

## 参考

- https://github.com/oauth2-proxy/oauth2-proxy
- https://qiita.com/convto/items/2822d029349cb1b4df93
- https://qiita.com/OmeletteCurry19/items/f24ee02a942d8f6931a5
