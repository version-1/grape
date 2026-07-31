# `grape init`

Creates `~/.grape/grape.json` from `grape.example.json` in the current directory.

```sh
grape init
```

Set `GRAPE_HOME` to create `$GRAPE_HOME/grape.json` instead. The command creates its destination directory when necessary and refuses to overwrite an existing config file.

## Migrating from gw

This release does not automatically read the former `gw` locations. Rename `gw.json` to `grape.json`, replace `GW_HOME` with `GRAPE_HOME`, and move `~/.gw` to `~/.grape`. Pass `--config` to use a config file at another path.
