# gitstatus

Prints a one-line summary of a git repository for a shell prompt to consume.

A drop-in replacement for the `gitstatus.py` that ships with [oh-my-zsh]'s
`git-prompt` plugin, in Go, so the prompt does not pay Python interpreter
startup on every command and every `cd`.

Not to be confused with [romkatv/gitstatus], a different and more ambitious
project. That one is a long-lived daemon that walks the index itself; this is a
single-shot command that shells out to `git` exactly as the script it replaces
does.

## Install

```bash
brew install icco/tap/gitstatus
```

Or `go install github.com/icco/gitstatus@latest`.

## Use

```console
$ gitstatus
main 0 0 0 0 1 0 1 0 0
```

Ten space-separated fields, no trailing newline:

| # | field | meaning |
|---|-------|---------|
| 1 | branch | branch name, or the tag/short hash when HEAD is detached |
| 2 | ahead | commits ahead of upstream |
| 3 | behind | commits behind upstream |
| 4 | staged | staged paths |
| 5 | conflicts | unmerged paths |
| 6 | changed | paths modified in the worktree |
| 7 | untracked | untracked paths |
| 8 | stashed | stash entries |
| 9 | clean | 1 when nothing above is dirty, else 0 |
| 10 | deleted | paths deleted in the worktree |

Outside a repository it prints nothing and exits 0.

### With oh-my-zsh's git-prompt

Point `update_current_git_vars` at the binary, keeping the script as a fallback
so the prompt still works on machines where the binary is not installed:

```zsh
if (( $+commands[gitstatus] )); then
  _GIT_STATUS=$(gitstatus 2>/dev/null)
else
  _GIT_STATUS=$(python3 "$__GIT_PROMPT_DIR/gitstatus.py" 2>/dev/null)
fi
```

## Compatibility

Output is byte-for-byte identical to `gitstatus.py`, verified by a differential
test that runs both over fourteen repository states (clean, staged, conflicted,
detached at one and two tags, stashed, ahead of upstream, no commits yet, and
so on):

```bash
task test:python
```

That fidelity includes the script's quirks, which are reproduced deliberately
rather than fixed, because the point is for the prompt's numbers not to change:

- A staged-and-modified path (`MM`) counts as **both** changed and staged; `MD`
  counts as both deleted and staged. The Y-column tests are independent `if`s
  in the original, not a chain.
- Only an `X` of `U` counts as a conflict, so `AU` and `DD` — which git also
  calls unmerged — are counted as staged instead.
- `clean` ignores stashes: a repository with stashes and no other changes is
  still clean.

Two behaviours differ, both cases where the original crashes and prints
nothing. This prints nothing too, so the prompt sees no difference:

- A branch whose name contains `...` raises `ValueError` in the original.
- Status output with no `##` header raises `NameError`.

One behaviour is deliberately better: each `git` call is bounded by a two
second timeout. The prompt runs on every command, so a git that hangs on a
stalled network filesystem degrades to "no git info" instead of freezing the
shell. The original would wait as long as git did.

## Speed

`go test -bench`, on an M5 MacBook Air. The two `Process` benchmarks are the
comparison that matters: both fork a fresh process, which is what the prompt
does on every `precmd` and every `chpwd`.

```
BenchmarkBinaryProcess-10          48    24196661 ns/op    71985 B/op    57 allocs/op
BenchmarkPythonProcess-10          24    49639413 ns/op    72032 B/op    58 allocs/op
BenchmarkRender-10                 73    17749821 ns/op   152108 B/op   199 allocs/op
BenchmarkParse-10             4747642         260.2 ns/op     376 B/op     6 allocs/op
```

So about **25 ms saved per invocation**, or ~50 ms per command once the prompt
has called it twice.

The interesting part is `BenchmarkParse` at 260 **nanoseconds**. Parsing was
never the cost, in either language — it is five orders of magnitude below the
total. The 25 ms this saves is Python interpreter startup, and the 17.7 ms
floor `BenchmarkRender` still pays is forking `git`, which this does exactly as
often as the script did.

Cutting below that floor means not shelling out to `git` at all, which is what
[romkatv/gitstatus] does.

## Develop

```bash
task            # build, vet, lint, test
task test       # tests with race detection and coverage
task test:python  # differential test against gitstatus.py
task bench      # benchmarks, including vs gitstatus.py
```

## License

MIT. `gitstatus.py`, whose behaviour this reproduces, is part of [oh-my-zsh] and
also MIT licensed.

[oh-my-zsh]: https://github.com/ohmyzsh/ohmyzsh/tree/master/plugins/git-prompt
[romkatv/gitstatus]: https://github.com/romkatv/gitstatus
