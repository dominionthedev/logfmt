<p align="center">
  <img src="./assets/logo.svg" alt="logfmt logo" width="600" />
</p>

# logfmt 🪵

[![Build Status](https://img.shields.io/github/actions/workflow/status/dominionthedev/logfmt/go.yml?branch=main)](https://github.com/dominionthedev/logfmt/actions/workflows/go.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/dominionthedev/logfmt)](./go.mod)
[![License](https://img.shields.io/github/license/dominionthedev/logfmt)](./LICENSE)

Pipe logs and get formatted, colorized, filterable terminal output.  
Supports **JSON** and **logfmt** (`key=value`) style logs. Falls back gracefully for plain text.

## Install

```bash
go install github.com/dominionthedev/logfmt@latest
```

Or build locally:

```bash
git clone ...
cd logfmt
go build -o logfmt .
```

## Usage

```bash
# Basic — pipe anything in
tail -f app.log | logfmt

# Filter to warn and above
cat app.log | logfmt --level warn

# Only show lines matching a string
kubectl logs my-pod | logfmt --filter "user_id=42"

# Hide KV pairs, show time + level + message only
cat app.log | logfmt --time-only

# Combine
tail -f app.log | logfmt --level error --filter "database"
```

## Flags

| Flag          | Short | Description                                               |
| ------------- | ----- | --------------------------------------------------------- |
| `--filter`    | `-f`  | Only show lines containing this string (case-insensitive) |
| `--level`     | `-l`  | Minimum level: `debug`, `info`, `warn`, `error`, `fatal`  |
| `--time-only` | `-t`  | Hide KV pairs, show time + level + message only           |
| `--no-color`  |       | Disable color output                                      |

## Supported formats

- **JSON** — detects `{` prefix, reads common keys (`level`, `msg`, `time`, etc.)
- **logfmt** — parses `key=value` and `key="quoted value"` pairs
- **Plain text** — rendered as-is in muted style

## Level colors

| Level | Color           |
| ----- | --------------- |
| DEBUG | Blue            |
| INFO  | Green           |
| WARN  | Amber           |
| ERROR | Red             |
| FATAL | Red + underline |
