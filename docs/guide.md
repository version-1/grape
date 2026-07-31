# Grape Guide

## Configuration File Priority

When a command accepts `--config` (or `-c`), grape resolves `grape.json` in the following order:

1. The path explicitly passed with `--config` or `-c`.
2. `./grape.json` in the current working directory.
3. `$GRAPE_HOME/grape.json` when `GRAPE_HOME` is set.
4. `~/.grape/grape.json` when `GRAPE_HOME` is not set.

An explicit config path is used as provided. When no path is specified, a config in the current working directory takes precedence over the user-level config.
