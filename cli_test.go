package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIVersion(t *testing.T) {
	t.Parallel()
	for _, arg := range []string{"-v", "--version", "version"} {
		var out, errOut bytes.Buffer
		if code := cli([]string{arg}, &out, &errOut); code != 0 {
			t.Errorf("cli(%q) exit = %d, want 0", arg, code)
		}
		if !strings.HasPrefix(out.String(), "gitstatus ") {
			t.Errorf("cli(%q) stdout = %q, want a version line", arg, out.String())
		}
		if errOut.Len() != 0 {
			t.Errorf("cli(%q) wrote to stderr: %q", arg, errOut.String())
		}
	}
}

func TestCLIHelp(t *testing.T) {
	t.Parallel()
	for _, arg := range []string{"-h", "--help", "help"} {
		var out, errOut bytes.Buffer
		if code := cli([]string{arg}, &out, &errOut); code != 0 {
			t.Errorf("cli(%q) exit = %d, want 0", arg, code)
		}
		if !strings.Contains(out.String(), "usage: gitstatus") {
			t.Errorf("cli(%q) stdout = %q, want usage", arg, out.String())
		}
	}
}

func TestCLIUnknownArg(t *testing.T) {
	t.Parallel()
	var out, errOut bytes.Buffer
	if code := cli([]string{"--nope"}, &out, &errOut); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "unknown argument") {
		t.Errorf("stderr = %q, want an unknown-argument message", errOut.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty on error", out.String())
	}
}

func TestCLINoArgsWritesStatusLine(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	commitFile(t, dir, "a.txt", "a\n")
	t.Chdir(dir)

	var out, errOut bytes.Buffer
	if code := cli(nil, &out, &errOut); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if got, want := out.String(), "main 0 0 0 0 0 0 0 1 0"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	// The prompt splits on spaces; a trailing newline would corrupt the last field.
	if strings.HasSuffix(out.String(), "\n") {
		t.Error("stdout ends with a newline")
	}
}

func TestGitReportsFailure(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// A subcommand git does not have must surface as an error, which is how
	// render() distinguishes "not a repository" from a usable status.
	if _, err := git("definitely-not-a-subcommand"); err == nil {
		t.Error("git() returned no error for an invalid subcommand")
	}
}

func TestGitPassesLangC(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	commitFile(t, dir, "a.txt", "a\n")
	t.Chdir(dir)

	// LANG=C keeps the branch header in the English form parseHeader matches.
	out, err := git("status", "--porcelain", "--branch")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(out), "## main") {
		t.Errorf("status header = %q, want it to start with %q", string(out), "## main")
	}
}
