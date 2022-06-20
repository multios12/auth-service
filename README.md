# 認証サービス

## URL
* /auth/login  - GET  - ログインページを表示する。ログインに成功した場合、"/"にリダイレクトする
* /auth/logout - GET  - 認証情報をクリアし、ログインページにリダイレクトする
* /auth/start  - POST - 認証処理を行うAPI
* /auth/auth   - GET  - nginx auth_requestモジュールのためのAPI。レスポンスコード202 authorizedまたは、401 unauthorizedを返す。
* /auth/info   - GET  - ユーザ情報を返すAPI
* /auth/info   - POST - ユーザ情報を設定するAPI

## 参考
https://github.com/oauth2-proxy/oauth2-proxy
https://qiita.com/convto/items/2822d029349cb1b4df93
https://qiita.com/OmeletteCurry19/items/f24ee02a942d8f6931a5

-------------------------------------------------------------
## 開発環境の立ち上げ方法

必須ソフトウェア：VS Code, Docker Desktop
  ※VS Codeに[Remote - Containers] [Remote Development]Extensionをインストール
Vscode上でCTRL+SHIFT+P押下、[Reopen in Container]選択で開発環境の立ち上げが可能

### 実行手順
1. 実行とデバッグで「go API Server」を選択、実行
2. 実行とデバッグで「debug react」を選択、実行
   ※react上で、react開発サーバが立ち上がっているため、通常2は不要

-------------------------------------------------------------
## create new react project
> create-react-app --template typescript
> yarn add bulma
