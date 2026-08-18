# 開発者向け情報

## 開発環境

必要なもの:

- Rust stable（edition 2024 対応）
- または、VS Code、Docker Desktop、VS Code の Dev Containers 拡張機能

VS Codeではコマンドパレットから `Dev Containers: Reopen in Container` を選択すると、リポジトリの `.devcontainer` を使った開発環境を起動できます。

## ソースからの起動

プロジェクトルートに `setting.json` を配置して、次のコマンドを実行します。

```bash
cargo run
```

サービスは `0.0.0.0:3000` で待ち受けます。開発コンテナでは動作確認のため3000番をホストへ公開します。この公開設定は開発専用であり、本番コンテナには適用しません。

## パスワードハッシュ生成ツール

開発環境ではCargoから実行できます。

```bash
cargo run --manifest-path generate_password_hash/Cargo.toml -- 'your-password'
```

## フォーマット・静的解析・テスト

本体を確認します。

```bash
cargo fmt --all -- --check
cargo clippy --all-targets --all-features --locked -- -D warnings
cargo test --all-features --locked
```

`generate_password_hash` も独立したCargoプロジェクトとして確認します。

```bash
cargo fmt --manifest-path generate_password_hash/Cargo.toml --all -- --check
cargo clippy --manifest-path generate_password_hash/Cargo.toml --all-targets --all-features --locked -- -D warnings
cargo test --manifest-path generate_password_hash/Cargo.toml --all-features --locked
```

## リリースビルド

```bash
cargo build --all-features --locked --release
cargo build --manifest-path generate_password_hash/Cargo.toml --all-features --locked --release
```

成果物は次の場所に生成されます。

```text
target/release/auth-service
generate_password_hash/target/release/generate_password_hash
```

配布用のmusl静的リンクバイナリは、専用のAlpineビルダーで生成します。

```bash
docker build \
  --file Dockerfile.musl \
  --target artifact \
  --output type=local,dest=dist \
  .
```

成果物は `dist/auth-service` と `dist/generate_password_hash` に生成されます。OpenSSLはAlpineの静的ライブラリを使用し、musl libcとともに静的リンクされます。

### musl版のリリース前確認

Dockerを利用できる開発端末では、GitHubへpushする前に上記のビルドを実行し、生成されたファイルを確認します。

```bash
file dist/auth-service dist/generate_password_hash
readelf -d dist/auth-service
readelf -d dist/generate_password_hash
```

`file` の結果が静的リンクを示し、`readelf -d` の結果に共有ライブラリを表す `NEEDED` が存在しないことを確認します。次のコマンドが何も出力せず、終了コード `1` になるのが正常です。

```bash
readelf -d dist/auth-service | grep NEEDED
readelf -d dist/generate_password_hash | grep NEEDED
```

auth-serviceがAlpine上で起動することは、次のように確認できます。

```bash
docker run --rm \
  -v "$PWD/dist/auth-service:/usr/local/bin/auth-service:ro" \
  -v "$PWD/setting.json:/work/setting.json:ro" \
  -w /work \
  alpine:3.22 \
  /usr/local/bin/auth-service
```

起動ログに `event=service_started` が出力されることを確認し、確認後は `Ctrl+C` で停止します。この確認では設定を更新しないため `setting.json` を読み取り専用でマウントします。パスキー登録・認証を確認する場合は、atomic renameを可能にするため設定ファイルを含むディレクトリ全体を読み書き可能でマウントします。

GitHub Actionsの実環境でCIを確認する場合は、`main` 以外の検証ブランチへpushします。

```bash
git switch -c test/musl-ci
git push -u origin test/musl-ci
```

CIは全ブランチへのpushで起動します。Release workflowは `main` へのpushかつ `Cargo.toml` が変更された場合だけ起動するため、検証ブランチからReleaseは作成されません。

GitHub CLIを利用する場合は、実行状況と失敗ログを次のコマンドで確認できます。

```bash
gh run list --branch test/musl-ci
gh run watch
gh run view --log-failed
```

## GitHub Actions

`.github/workflows/ci.yml` は、すべてのブランチへのpush、Pull Request、手動実行で起動します。本体と `generate_password_hash` に対して、GNU環境でフォーマット、Clippy、テストを実行し、配布用のmusl静的リンク版をReleaseビルドします。GNU版のReleaseバイナリは生成しません。

`.github/workflows/release.yml` は、`main` ブランチの `Cargo.toml` が変更されたときに起動します。バージョンが直前の値より上がっていることを確認し、次の処理を実行します。

1. Alpineビルダーで本体と `generate_password_hash` のmusl静的リンク版をReleaseビルドします。
2. Linux x86_64向けのアーカイブとSHA-256ファイルを作成します。
3. `vX.Y.Z` タグとGitHub Releaseを作成します。
4. 作成した4ファイルをRelease Assetsへ登録します。

リリースする場合は、`Cargo.toml` のバージョンを `x.y.z` 形式で上げてから `main` へマージします。

```toml
[package]
name = "auth-service"
version = "2.1.0"
```
