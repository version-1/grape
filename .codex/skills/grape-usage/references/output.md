# Output and Color

Grape-generated diagnostics use stderr:

```text
grape: <information>
grape: warning: <warning>
grape: <error>
```

Only warning and error prefixes are colored, and only when stderr is a terminal. Tables, previews, prompts, and progress use an independent stdout terminal check. Pipes, redirects, and `NO_COLOR` disable color for the relevant output. Raw Git subprocess streams are passed through without recoloring.

After a successful `reset`, grape prints a `Branches` heading followed by every remaining local branch. Reset does not print grape-generated per-operation progress while removing worktrees, deleting branches, or adding configured worktrees.
