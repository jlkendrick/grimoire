# Grimoire — CLAUDE.md

## Project Overview

Grimoire is a declarative, language-agnostic execution framework written in Go. It translates pure, uninstrumented functions into fully typed CLI commands (and eventually REST APIs) using YAML configuration files. There is a hybrid local/global architecture: a global "grimoire" (spellbook) for personal automation scripts, and repo-level `grim.yaml` files ("spells") for team-shared commands.

The CLI binary is currently named `sigil` (`rootCmd.Use = "sigil"`), though the project and compiled binary are named `grimoire`. This naming is in flux.

## Tech Stack

- **Language**: Go 1.23.0
- **CLI framework**: [Cobra](https://github.com/spf13/cobra)
- **YAML parser**: `github.com/goccy/go-yaml` (supports comments, used for config roundtrips)
- **TOML parser**: `github.com/pelletier/go-toml/v2`
- **Code analysis**: `github.com/smacker/go-tree-sitter` + Python grammar (AST-based signature extraction)
- **Target runtime**: Python 3.x (only language supported currently)

## Build & Test

```sh
go build -o grimoire .     # build binary
go test ./...               # run all tests
go test ./cmd/...           # run cmd tests only
go test -v ./core/...       # verbose core tests
```

## Key Architecture

### Static vs. Dynamic Commands

Commands are split into two categories:

- **Static commands** (`init`, `add`, `sync`, `register`, `clean`): always registered, no config required
- **Dynamic commands**: generated at startup from `grim.yaml` — one `cobra.Command` per function entry

`root.go` skips config loading entirely when a static command is detected (checked via `os.Args[1]`), which keeps startup fast and avoids errors when no config exists.

### Config Loading & Caching

`core/config.go` implements `LoadConfig(scope string)`:

- `"local"`: walks upward from cwd looking for `grim.yaml`
- `"global"`: loads `~/.grimoire/config.yaml` (**currently hardcoded to** `~/Code/Projects/grimoire/.grimoire/config.yaml` — known TODO, appears in 3 places)
- Config is cached in a package-level variable after first load; call `core.ResetConfigCache()` in tests to isolate state

### Function Execution Pipeline

`core/interface.go` → `core.ExecuteFunction(function, argsMap)`:

1. Assigns a runtime adapter based on file extension (`.py` → `PythonAdapter`)
2. Adapter resolves the interpreter (explicit path → `.venv` → `pyproject.toml` → `requirements.txt` → system `python`)
3. Generates an inline Python script that uses `importlib` to load the module and calls the function with `**kwargs`
4. Serializes `argsMap` to JSON, pipes it to the subprocess stdin
5. Returns stdout bytes; stderr is wrapped into a formatted error

### Python Environment Provisioning

`core/runtimes/python.go` auto-provisions virtual environments:

- Caches envs in `~/.grimoire/envs/{sha256_of_deps_file}/`
- Tracks freshness via a `.sigil_req_hash` sentinel file
- Auto-rebuilds if dependency file content changes
- Supports both `requirements.txt` and `pyproject.toml`

### Argument Type System

`types/types.go` defines `Arg` with name, type string (`"int"`, `"str"`, `"bool"`, `"float"`), and default. Two special methods:

- `UnmarshalYAML()` — normalizes numeric types that YAML parses as floats/ints
- `CastAndSetDefault()` — converts string defaults to proper Go types after loading

### Python Signature Extraction

`parsers/python.go` uses tree-sitter to parse Python source and extract function signatures. Handles typed params, defaults, `*args`, and `**kwargs`. Used by the `add` command to auto-populate `args` in `grim.yaml`.

## Directory Structure

```
cmd/          CLI commands (root, init, add, sync, commands)
core/         Execution engine and config loading
  runtimes/   Language-specific runtime adapters
config/       YAML config parsing and generation
parsers/      Language-specific code analysis (tree-sitter)
types/        Shared data structures (Config, Function, Arg)
utils/        File utilities (path expansion, hashing, traversal)
sample/       Example project with grim.yaml and Python scripts
```

## Known Issues & TODOs

- **Hardcoded global config path**: `~/.grimoire/config.yaml` is hardcoded to the dev machine path in 3 places — needs to use `os.UserHomeDir()` universally
- **`register` and `clean` commands**: stubs only, not implemented
- **Only Python supported**: `assignAdapter` returns an error for any non-`.py` file extension
- **CLI name mismatch**: binary is `grimoire`, but `rootCmd.Use` is `sigil`
- **REST API generation**: planned but not started

## grim.yaml Schema

```yaml
functions:
  - name: cli_command_name       # name of the generated CLI subcommand
    path: path/to/file.py        # path relative to the grim.yaml file
    function: python_function    # name of the Python function to call
    interpreter: /path/to/python # optional: explicit interpreter path
    args:
      - name: param_name
        type: str | int | bool | float
        default: optional_value
```

Config files are searched by walking upward from cwd. Global config lives at `~/.grimoire/config.yaml` (uses absolute paths).
