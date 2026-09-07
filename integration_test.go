package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoState builds a git repository in a fresh temp dir and chdirs into it.
// Each returned name is a scenario the prompt has to render correctly.
type repoState struct {
	name  string
	setup func(t *testing.T, dir string)
	// sub, when set, is the subdirectory of dir to run in -- used to enter a
	// linked worktree rather than the main checkout.
	sub  string
	want string
}

// dirFor returns the directory a scenario should run in.
func (r repoState) dirFor(dir string) string {
	if r.sub == "" {
		return dir
	}
	return filepath.Join(dir, r.sub)
}

// gitCmd builds a git command isolated from the ambient configuration, with
// an identity of its own. Every git call in these tests must go through here:
// a call that inherits the developer's ~/.gitconfig passes locally and then
// fails on a CI runner that has no user.email configured.
func gitCmd(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
	)
	return cmd
}

func gitDo(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := gitCmd(dir, args...).CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// gitExpectFail runs a git command that is meant to exit non-zero, and fails
// the test if it unexpectedly succeeds -- otherwise a scenario meant to build
// a conflict silently produces a clean repository instead.
func gitExpectFail(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := gitCmd(dir, args...).CombinedOutput(); err == nil {
		t.Fatalf("git %s: expected a non-zero exit, got success\n%s",
			strings.Join(args, " "), out)
	}
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	gitDo(t, dir, "init", "-q", "-b", "main")
}

func commitFile(t *testing.T, dir, name, body string) {
	t.Helper()
	write(t, dir, name, body)
	gitDo(t, dir, "add", name)
	gitDo(t, dir, "commit", "-qm", "add "+name)
}

func states() []repoState {
	return []repoState{
		{"clean repo", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "a\n")
		}, "", "main 0 0 0 0 0 0 0 1 0"},

		{"untracked file", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "a\n")
			write(t, d, "new.txt", "n\n")
		}, "", "main 0 0 0 0 0 1 0 0 0"},

		{"modified worktree", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "a\n")
			write(t, d, "a.txt", "changed\n")
		}, "", "main 0 0 0 0 1 0 0 0 0"},

		{"staged change", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "a\n")
			write(t, d, "a.txt", "changed\n")
			gitDo(t, d, "add", "a.txt")
		}, "", "main 0 0 1 0 0 0 0 0 0"},

		{"staged then modified again", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "a\n")
			write(t, d, "a.txt", "staged\n")
			gitDo(t, d, "add", "a.txt")
			write(t, d, "a.txt", "and again\n")
		}, "", "main 0 0 1 0 1 0 0 0 0"},

		{"deleted in worktree", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "a\n")
			if err := os.Remove(filepath.Join(d, "a.txt")); err != nil {
				t.Fatal(err)
			}
		}, "", "main 0 0 0 0 0 0 0 0 1"},

		{"one stash", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "a\n")
			write(t, d, "a.txt", "changed\n")
			gitDo(t, d, "stash")
		}, "", "main 0 0 0 0 0 0 1 1 0"},

		{"two stashes", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "a\n")
			for _, s := range []string{"one", "two"} {
				write(t, d, "a.txt", s+"\n")
				gitDo(t, d, "stash")
			}
		}, "", "main 0 0 0 0 0 0 2 1 0"},

		{"no commits yet", func(t *testing.T, d string) {
			initRepo(t, d)
		}, "", "main 0 0 0 0 0 0 0 1 0"},

		{"detached at a tag", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "a\n")
			gitDo(t, d, "tag", "v1.0.0")
			gitDo(t, d, "checkout", "-q", "v1.0.0")
		}, "", "v1.0.0 0 0 0 0 0 0 0 1 0"},

		{"detached with two tags", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "a\n")
			gitDo(t, d, "tag", "v1.0.0")
			gitDo(t, d, "tag", "v2.0.0")
			gitDo(t, d, "checkout", "-q", "v2.0.0")
		}, "", "v2.0.0+ 0 0 0 0 0 0 0 1 0"},

		{"branch with a slash", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "a\n")
			gitDo(t, d, "checkout", "-q", "-b", "feat/thing")
		}, "", "feat/thing 0 0 0 0 0 0 0 1 0"},

		{"ahead of upstream", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "a\n")
			remote := t.TempDir()
			gitDo(t, remote, "init", "-q", "--bare")
			gitDo(t, d, "remote", "add", "origin", remote)
			gitDo(t, d, "push", "-q", "-u", "origin", "main")
			commitFile(t, d, "b.txt", "b\n")
		}, "", "main 1 0 0 0 0 0 0 1 0"},

		// The original reaches for --git-common-dir rather than --git-dir
		// precisely so a linked worktree reports the stashes it shares with
		// the main checkout instead of zero. This is the only scenario that
		// exercises that choice.
		{"linked worktree shares stashes", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "a\n")
			write(t, d, "a.txt", "changed\n")
			gitDo(t, d, "stash")
			gitDo(t, d, "worktree", "add", "-q", "-b", "wt", filepath.Join(d, "wt"))
		}, "wt", "wt 0 0 0 0 0 0 1 1 0"},

		{"conflict", func(t *testing.T, d string) {
			initRepo(t, d)
			commitFile(t, d, "a.txt", "base\n")
			gitDo(t, d, "checkout", "-q", "-b", "other")
			write(t, d, "a.txt", "other\n")
			gitDo(t, d, "commit", "-aqm", "other")
			gitDo(t, d, "checkout", "-q", "main")
			write(t, d, "a.txt", "main\n")
			gitDo(t, d, "commit", "-aqm", "main")
			// The conflicting merge. Asserting it fails is what catches an
			// environment where git refuses for an unrelated reason.
			gitExpectFail(t, d, "merge", "other")
		}, "", "main 0 0 0 1 0 0 0 0 0"},
	}
}

func TestRenderAgainstRealRepos(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	for _, st := range states() {
		t.Run(st.name, func(t *testing.T) {
			dir := t.TempDir()
			st.setup(t, dir)
			t.Chdir(st.dirFor(dir))
			if got := render(); got != st.want {
				t.Errorf("render() = %q\n     want %q", got, st.want)
			}
		})
	}
}

func TestRenderOutsideRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	// A temp dir is not a repo, unless a parent happens to be one; guard by
	// checking git agrees before asserting.
	dir := t.TempDir()
	t.Chdir(dir)
	if err := gitCmd(dir, "rev-parse", "--git-dir").Run(); err == nil {
		t.Skip("temp dir is inside a git repository")
	}
	if got := render(); got != "" {
		t.Errorf("render() = %q outside a repository, want empty", got)
	}
}
