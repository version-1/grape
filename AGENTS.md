# AGENTS.md

All documentation must be written in English.

## プロジェクト概要

`gw` は `git worktree` ワークフローを支援する Go CLI です。エントリーポイントは `cmd/gw` で、コアの責務は `internal/app`、`internal/config`、`internal/ui`、`internal/worktree` に分離されています。

詳細な利用方法は `README.md` と `REFERENCE.md` を参照してください。

## 開発ルール

- Go 1.22 を使用する。
- 既存のパッケージ責務を維持し、CLI の制御は `internal/app`、Git 操作は `internal/worktree` に置く。
- 公開する CLI の引数、終了コード、設定ファイル形式を変更する場合は、互換性とドキュメントへの影響を確認する。
- `remove` と `reset` は worktree とローカルブランチを削除するため、対象判定と確認フローを弱めない。
- ユーザー向けの README は英語で記述する。コマンドの完全な仕様は `REFERENCE.md` に記載する。

## よく使うコマンド

```sh
# 全テスト
go test ./...

# ローカルバイナリのビルド
make build

# CLI を直接実行
go run ./cmd/gw list
```

変更後は、影響するパッケージのテストに加えて `go test ./...` を実行してください。

## 変更時の確認事項

- Go コードは `gofmt` を適用する。
- CLI の振る舞いを変えた場合は、対応するテストと `README.md`／`REFERENCE.md` を更新する。
- 削除を伴う変更では、main working tree とそのチェックアウト中のブランチが対象外であることを確認する。
