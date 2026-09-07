# gitstatus

Prints a git repository's state as one line, for a shell prompt to consume.

A drop-in replacement for the `gitstatus.py` in [oh-my-zsh]'s `git-prompt` plugin, in Go, so the prompt stops paying Python interpreter startup on every command and every `cd`.

Not to be confused with [romkatv/gitstatus], a different and better-known project: that one is a long-lived daemon that walks the index itself, this is a single-shot command that shells out to `git` exactly as the script it replaces does.

## Install

```bash
brew install icco/tap/gitstatus        # macOS; casks are macOS-only
go install github.com/icco/gitstatus@latest
```

## Use

```console
$ gitstatus
main 0 0 0 0 1 0 1 0 0
```

Ten space-separated fields, no trailing newline:

```
branch ahead behind staged conflicts changed untracked stashed clean deleted
```

`clean` is 1 when nothing else is dirty, else 0. Outside a repository it prints nothing and exits 0.

To use it from oh-my-zsh's `git-prompt`, override `update_current_git_vars` to call `gitstatus` instead of `python3 gitstatus.py`, keeping the script as a fallback for machines without the binary.

## Compatibility

Output is byte-for-byte identical to `gitstatus.py`, verified by a differential test over fifteen repository states — clean, staged, conflicted, detached, stashed, ahead of upstream, linked worktree, and more (`task test:python`).

The script's quirks are reproduced deliberately, not fixed, so the prompt's numbers do not change:

- `MM` counts as both changed and staged, and `MD` as both deleted and staged.
- Only an `X` of `U` is a conflict, so `AU` and `DD` count as staged instead.
- `clean` ignores stashes.

Two things differ, both cases where the original raises and prints nothing — a branch name containing `...`, and status output with no `##` header. This prints nothing too, so the prompt sees no difference. One thing is deliberately better: each `git` call has a two second timeout, so a stalled filesystem degrades to no output rather than freezing the shell.

## Speed

About 25 ms saved per call, all of it Python startup. Parsing is 260 ns and has never been the bottleneck; the remaining cost is forking `git`. See [#1](https://github.com/icco/gitstatus/pull/1) for why go-git is not faster.

## Develop

`task` builds, vets, lints and tests. See [AGENTS.md](AGENTS.md).

## License

MIT. `gitstatus.py`, whose behaviour this reproduces, is part of [oh-my-zsh] and also MIT.

[oh-my-zsh]: https://github.com/ohmyzsh/ohmyzsh/tree/master/plugins/git-prompt
[romkatv/gitstatus]: https://github.com/romkatv/gitstatus
