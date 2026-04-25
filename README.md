# Grimoire

> *A spellbook for your codebase.*

**Status: Work in progress — not yet ready for general use.**

---

Grimoire is a declarative, language-agnostic execution framework. The core idea is simple: you write pure business logic in whatever language you like, and Grimoire translates it into a fully typed CLI — no boilerplate, no argument parsing, no plumbing. Point it at a function, describe the interface in YAML, and it handles the rest.

Later, the same configuration will also generate REST APIs from the same functions, with no changes to your code.

## The Metaphor

Grimoire is your personal spellbook. Every developer has random utility scripts scattered across their system — one-off data transforms, deployment helpers, local tools that only work on their machine. Grimoire brings them together into a single, unified interface.

A `scroll.yaml` file is a *spell*: a precise recipe that describes exactly how to invoke the functions in a specific repo. Unlike a generic script runner, a spell carries the full incantation — the function to call, the arguments it expects, the types, the defaults. Anyone with Grimoire installed can pick up a spell and cast it.

## Architecture

Grimoire has a hybrid local/global design.

**The global grimoire** (`~/.grimoire/`) is your personal spellbook. Register functions from anywhere on your system — old shell scripts, Python utilities, tools you've written over the years — and invoke them from anywhere, under a single unified CLI.

**Repo-level spells** (`scroll.yaml`) are committed alongside your code. Push a `scroll.yaml` to a shared repo and any teammate with Grimoire installed can immediately run the functions it describes, with full argument type-checking and automatic dependency provisioning — no setup instructions, no "it works on my machine."

## How It Works

A `scroll.yaml` declares the functions you want to expose:

```yaml
functions:
  - name: greet
    path: scripts/greet.py
    function: say_hello
    args:
      - name: name
        type: str
      - name: times
        type: int
        default: 1
```

From this, Grimoire generates a CLI command:

```sh
grimoire greet --name "Alice" --times 3
```

Grimoire handles interpreter resolution (virtual environments, `pyproject.toml`, `requirements.txt`, or system Python), argument parsing, type coercion, and execution. Your function stays completely uninstrumented — no imports, no decorators, no framework code.

## Key Commands


| Command                          | Description                                                                                                              |
| -------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `grimoire init`                  | Scaffold a `scroll.yaml` in the current directory                                                                         |
| `grimoire add <file>:<function>` | Add a function to `scroll.yaml` and auto-extract its signature                                                            |
| `grimoire sync`                  | Regenerate argument signatures for all registered functions                                                              |
| `grimoire register [path]`       | Register a project's `scroll.yaml` with the global grimoire (defaults to nearest `scroll.yaml` found via upward traversal) |
| `grimoire clean [--global]`      | Remove cached venvs for functions whose source files no longer exist                                                     |
| `grimoire <name> [flags]`        | Run a function by its declared name                                                                                      |


## Project Structure

```
cmd/          CLI layer — command registration and flag generation
core/         Execution engine — runtime dispatch and config loading
  runtimes/   Language adapters (Python, Go)
config/       YAML parsing and config generation
parsers/      Source code analysis for signature extraction (tree-sitter)
types/        Shared data structures
utils/        File utilities
sample/       Example project with scroll.yaml and Python scripts
```

## Current Language Support

- **Python** — full support, including automatic virtual environment provisioning from `requirements.txt` or `pyproject.toml`
- **Go** — full support; Grimoire creates an isolated wrapper module per project, compiles it on first use, and caches the binary for fast subsequent invocations

More runtimes are planned. The adapter interface is designed to be language-agnostic from the start.

## Installation

**Homebrew (recommended):**

```sh
brew tap jlkendrick/tap
brew install grimoire
```

**`go install`:**

```sh
go install github.com/jlkendrick/grimoire@latest
```

> **Note:** Grimoire uses [tree-sitter](https://tree-sitter.github.io/) for source code analysis, which requires cgo and a C compiler (`gcc` or `clang`). Make sure one is installed before building.
>
> On macOS: `xcode-select --install`
> On Linux: `sudo apt install gcc` (or your distro's equivalent)

## Development

```sh
go build -o grimoire .
go test ./...
```

Requires Go 1.23+ and a C compiler (see Installation note above).