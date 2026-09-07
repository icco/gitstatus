// Command gitstatus prints a one-line summary of the current git repository
// for a shell prompt to consume.
//
// Output is ten space-separated fields with no trailing newline:
//
//	branch ahead behind staged conflicts changed untracked stashed clean deleted
//
// It is a drop-in replacement for the gitstatus.py that ships with oh-my-zsh's
// git-prompt plugin and reproduces that script's field semantics exactly,
// quirks included -- see the comment in parse. Outside a repository, or in one
// whose status carries no branch header, it prints nothing and exits 0, which
// is what the plugin treats as "no git info here".
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Set by the linker at release time; see .goreleaser.yaml.
var (
	Version   = "dev"
	CommitSHA = "none"
)

func main() {
	os.Exit(cli(os.Args[1:], os.Stdout, os.Stderr))
}

// cli handles arguments and writes the result, returning the process exit
// code. Split out from main so the argument handling is testable.
func cli(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "-v", "--version", "version":
			fmt.Fprintf(stdout, "gitstatus %s (%s)\n", Version, CommitSHA)
			return 0
		case "-h", "--help", "help":
			fmt.Fprint(stdout, usage)
			return 0
		default:
			fmt.Fprintf(stderr, "gitstatus: unknown argument %q\n\n%s", args[0], usage)
			return 2
		}
	}

	fmt.Fprint(stdout, render())
	return 0
}

// render produces the line main prints. It is empty outside a git repository,
// and empty when the status output carries no branch header.
func render() string {
	// LANG=C so the branch header stays parseable under a localised git.
	out, err := git("status", "--porcelain", "--branch")
	if err != nil {
		return "" // Not a git repository.
	}

	s, ok := parse(out, tagnameOrHash)
	if !ok {
		return ""
	}
	s.Stashed = stashCount()
	return s.String()
}

const usage = `usage: gitstatus

Prints ten space-separated fields describing the git repository in the working
directory, with no trailing newline:

  branch ahead behind staged conflicts changed untracked stashed clean deleted

Prints nothing outside a git repository.

  -v, --version   print version and exit
  -h, --help      print this message and exit
`

// gitTimeout bounds each git call. The prompt runs this on every command and
// every directory change, so a git that hangs -- a stalled network
// filesystem, a pathologically large worktree -- has to degrade to "no git
// info" rather than freeze the shell. The script this replaces had no timeout
// and would hang for as long as git did.
const gitTimeout = 2 * time.Second

// git runs a git subcommand in the working directory and returns its stdout.
func git(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	// #nosec G204 -- every call site passes literals defined in this file.
	// Nothing from the environment, the arguments or the repository reaches
	// args, and the binary name is fixed.
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "LANG=C")
	return cmd.Output()
}

// tagnameOrHash names a detached HEAD: the highest-sorting tag pointing at it,
// suffixed with "+" when more than one does, else the short hash.
func tagnameOrHash() string {
	tagsOut, err := git(
		"for-each-ref",
		"--points-at=HEAD",
		"--count=2",
		"--sort=-version:refname",
		"--format=%(refname:short)",
		"refs/tags",
	)
	if err == nil {
		if tags := strings.Fields(string(tagsOut)); len(tags) > 0 {
			if len(tags) > 1 {
				return tags[0] + "+"
			}
			return tags[0]
		}
	}

	hashOut, err := git("rev-parse", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(hashOut))
}

// stashCount counts entries in the stash reflog. --git-common-dir rather than
// --git-dir so a linked worktree reports the stashes it actually shares
// instead of zero.
func stashCount() int {
	out, err := git("rev-parse", "--git-common-dir")
	if err != nil {
		return 0
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return 0
	}

	// #nosec G304 -- dir comes from `git rev-parse --git-common-dir`, and the
	// rest of the path is fixed. Opening it is the whole point.
	f, err := os.Open(filepath.Join(dir, "logs", "refs", "stash"))
	if err != nil {
		return 0 // No stash reflog means no stashes.
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	// Reflog lines carry two hashes, an identity and a message; the 64KiB
	// default is tight for a long stash message.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	n := 0
	for sc.Scan() {
		n++
	}
	if sc.Err() != nil {
		return 0
	}
	return n
}
