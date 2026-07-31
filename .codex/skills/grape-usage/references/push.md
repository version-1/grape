# `grape push`

Safely pushes the current symbolic branch to the fixed `origin` remote.

Only these forms are accepted:

```sh
grape push
grape push --force-with-lease
```

They execute exactly:

```text
git push origin HEAD:<current-branch>
git push --force-with-lease origin HEAD:<current-branch>
```

Other arguments, `--force`, `-f`, custom lease values, refspecs, tags, custom remotes, and multiple branches are rejected before Git push execution. Push does not prompt.

After strict global config validation, push requires a symbolic branch in a Git repository, rejects branch names beginning with `-`, validates the branch with Git, and rejects protected branch matches. It requires exactly one `remote.origin.pushurl`, or falls back to exactly one `remote.origin.url`.

Immediately before execution, grape logs the branch, selected origin URL, and exact command to stderr. Git subprocess streams pass through unchanged, and grape preserves the `git push` exit code.
