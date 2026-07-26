# gw Reference

`gw` は `git worktree` を扱うための Go CLI です。

標準の `git worktree` をそのまま呼び出す用途に加えて、worktree の見やすい一覧表示、branch からの逆引き、prefix / regex による削除、設定ファイルに基づく reset を提供します。

## 実行

```sh
cd shared/commands/gw
go run ./cmd/gw list
```

`gw` をビルドして PATH に置く場合は、entrypoint は `./cmd/gw` です。

```sh
make build
```

## Commands

### `gw help`

コマンド一覧、使い方、設定ファイルの読み込み順を表示します。

```sh
go run ./cmd/gw help
go run ./cmd/gw --help
go run ./cmd/gw -h
```

### `gw list`

worktree の path、branch、HEAD を色付きの表で表示します。

```sh
go run ./cmd/gw list
```

引数なしの `gw` も `gw list` と同じ動作です。

### `gw branch <branch-name>`

指定した local branch を参照している worktree を表示します。

```sh
go run ./cmd/gw branch feature/example
go run ./cmd/gw branch refs/heads/feature/example
```

`refs/heads/` prefix はあってもなくても同じ branch として扱います。該当する worktree がない場合は exit code `1` を返します。

### `gw remove <prefix>`

path が prefix 配下にある worktree を削除し、対応する local branch があれば `git branch -D` で削除します。

```sh
go run ./cmd/gw remove ../worktrees
```

prefix は絶対 path に正規化して比較します。prefix 自体と、その配下の worktree path が対象です。

### `gw remove --regex <pattern>`

worktree path に正規表現を適用し、match した worktree を削除します。対応する local branch があれば `git branch -D` で削除します。

```sh
go run ./cmd/gw remove --regex 'repo-feature-.+'
go run ./cmd/gw remove -r 'repo-feature-.+'
```

正規表現が不正な場合は exit code `2` を返します。

### `gw reset`

設定ファイルを読み、default branch 以外の worktree と local branch を削除してから、設定ファイルどおりに worktree を作成します。

```sh
go run ./cmd/gw reset
go run ./cmd/gw reset --config gw.json
go run ./cmd/gw reset -c gw.json
```

設定ファイルはデフォルトで `gw.json` を読みます。

`--config` を指定しない場合は、次の順序で設定ファイルを解決します。

1. カレントディレクトリの `gw.json`
2. `GW_HOME/gw.json`
3. `GW_HOME` が未設定の場合は `~/.gw/gw.json`

`reset` は破壊的操作です。default branch 以外の worktree と local branch を削除します。main working tree と main working tree が checkout している branch は削除しません。削除前に対象 worktree と branch を表示し、`y` または `yes` で確認された場合だけ処理を続行します。実行前に対象 repository と設定ファイルを確認してください。

## Reset Config

```json
{
  "default_branch": "main",
  "worktrees": [
    {
      "path": ".worktrees/1",
      "branch": "worktrees/1",
      "start_point": "origin/main"
    },
    {
      "path": ".worktrees/2",
      "branch": "worktrees/2",
      "start_point": "origin/main"
    },
    {
      "path": ".worktrees/3",
      "branch": "worktrees/3",
      "start_point": "origin/main"
    },
    {
      "path": ".worktrees/4",
      "branch": "worktrees/4",
      "start_point": "origin/main"
    },
    {
      "path": ".worktrees/5",
      "branch": "worktrees/5",
      "start_point": "origin/main"
    }
  ]
}
```

### `default_branch`

削除対象から除外する branch です。

省略した場合は `origin/HEAD` から検出します。

### `worktrees[].path`

作成する worktree の path です。

同じ path を複数指定することはできません。

### `worktrees[].branch`

作成する local branch 名です。

`refs/heads/` prefix はあってもなくても同じ branch として扱います。同じ branch を複数指定することはできません。

### `worktrees[].start_point`

`git worktree add -B <branch> <path> <start_point>` の `<start_point>` に渡す値です。

省略した場合は `default_branch` を起点にします。

## Unknown Commands

`list`、`branch`、`remove`、`reset` 以外の引数は `git worktree` にそのまま委譲します。

```sh
go run ./cmd/gw prune
go run ./cmd/gw list --porcelain
```

## Package Layout

```text
cmd/gw/              # entrypoint
internal/app/        # CLI dispatch and command orchestration
internal/config/     # reset config schema and validation
internal/ui/         # colored output and worktree table formatting
internal/worktree/   # git adapter, worktree parser, matcher
```
