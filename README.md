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
## デバッグ実行
vscode上での実行を前提。chromeを利用
フロントデバッグサーバ：ポート80
バックエンドサーバ：ポート3000

### 実行手順
1. 実行とデバッグで「go API Server」を選択、実行
2. 実行とデバッグで「debug react」を選択、実行

-------------------------------------------------------------
## nitialize setting
> npm i -g yarn create-react-app

## create new react project
> create-react-app --template typescript
> yarn add bulma axios
