# AGENTS.md

`gitstatus` prints a git repository's state as one line for a shell prompt. It is a drop-in replacement for the `gitstatus.py` in oh-my-zsh's `git-prompt` plugin.

## Commands

- `task` — build, vet, lint, test
- `task test` — tests with race detection and coverage
- `task test:python` — differential test against `gitstatus.py`
- `task bench` — benchmarks, including against the python script
- `task lint` — golangci-lint

`golangci-lint` fails locally when its own Go build predates the installed toolchain (`could not import io ... export data version 4`). Prefix it with the version in `go.mod`: `GOTOOLCHAIN=go1.26.2 golangci-lint run`. CI is unaffected because `setup-go` reads `go.mod`, which is also why the `go` directive must stay at a version the latest golangci-lint can build against.

## Architecture

One package, `main`:

- `status.go` — the `Status` struct (ten fields), `parse` (porcelain output → `Status`), and `parseHeader`. Calls no subprocesses, so the compatibility quirks are unit-testable.
- `main.go` — `cli` (arguments), `render` (the line `main` prints), and the git calls: `git`, `tagnameOrHash`, `stashCount`.

## Do not "fix" the quirks

Output must stay byte-for-byte identical to `gitstatus.py`, or the prompt's numbers change. These read as bugs and are deliberate:

- The Y-column tests in `parse` are independent `if`s rather than a chain, so `MM` counts as both changed and staged, and `MD` as both deleted and staged.
- Only an `X` of `U` counts as a conflict, so `AU` and `DD` — which git also calls unmerged — count as staged.
- `clean` ignores stashes.

`differential_test.go` enforces this against the real script. Run `task test:python` before and after touching `parse`.

## Performance

Parsing is 260 ns and is never the bottleneck; the cost is forking `git`, twice — `status --porcelain --branch`, then `rev-parse --git-common-dir` for the stash reflog. Eliminating that second fork is the open win.

Do not propose go-git. It was measured at 3-5x slower than forking on real worktrees, and v6 is 10x slower again, because go-billy adds several filesystem syscalls per path. See [#1](https://github.com/icco/gitstatus/pull/1).

## Tests

`integration_test.go` builds real repositories per scenario. Every git call must go through `gitCmd`, which isolates the ambient config and supplies an identity — a call that inherits a developer's `~/.gitconfig` passes locally and then fails in CI on a missing `user.email`.

CI enforces 80% coverage.

## Releasing

Push a `v*` tag, or let the daily cron compute the next version from conventional commits. goreleaser publishes the cask to `icco/homebrew-tap`, which needs the repo's `GH_PAT` secret. A failed run cannot be retried for the same tag, because GitHub releases are immutable — cut a new tag instead.
