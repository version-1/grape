# Output and Color

Grape-generated diagnostics use stderr:

```text
grape: <information>
grape: warning: <warning>
grape: <error>
```

Only warning and error prefixes are colored, and only when stderr is a terminal. Tables, previews, prompts, and progress use an independent stdout terminal check. Pipes, redirects, and `NO_COLOR` disable color for the relevant output. Raw Git subprocess streams are passed through without recoloring.
