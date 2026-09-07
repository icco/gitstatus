package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// benchRepo builds a small repository with something in every bucket, so the
// parser is not measured against a trivially clean status.
func benchRepo(b *testing.B) string {
	b.Helper()
	dir := b.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e",
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			b.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(name, body string) {
		if err := os.WriteFile(dir+"/"+name, []byte(body), 0o600); err != nil {
			b.Fatal(err)
		}
	}

	run("init", "-q", "-b", "main")
	write("a.txt", "a\n")
	run("add", "a.txt")
	run("commit", "-qm", "init")
	write("a.txt", "modified\n") // changed
	write("staged.txt", "s\n")
	run("add", "staged.txt")      // staged
	write("untracked.txt", "u\n") // untracked
	return dir
}

// BenchmarkRender measures the in-process cost: parsing plus the git
// subprocesses render() shells out to. This is the floor, and it excludes
// process startup, so it is not comparable to BenchmarkPythonProcess.
func BenchmarkRender(b *testing.B) {
	if _, err := exec.LookPath("git"); err != nil {
		b.Skip("git not installed")
	}
	b.Chdir(benchRepo(b))

	for b.Loop() {
		if render() == "" {
			b.Fatal("render() returned empty in a repository")
		}
	}
}

// BenchmarkParse isolates the parser from git entirely.
func BenchmarkParse(b *testing.B) {
	out := []byte("## main...origin/main [ahead 2, behind 1]\n" +
		" M a.txt\nA  staged.txt\n?? untracked.txt\nUU conflict.txt\n D gone.txt\n")
	detached := func() string { return "" }

	for b.Loop() {
		if _, ok := parse(out, detached); !ok {
			b.Fatal("parse reported no branch header")
		}
	}
}

// BenchmarkBinaryProcess and BenchmarkPythonProcess are the comparison that
// matters: both fork a fresh process, which is what the prompt does on every
// precmd and every chpwd. Build the binary first (task build).
func BenchmarkBinaryProcess(b *testing.B) {
	// Absolute: benchProcess sets cmd.Dir to the temp repo, so a relative
	// path would resolve against that instead of the package directory.
	bin, err := filepath.Abs("bin/gitstatus")
	if err != nil {
		b.Skip(err)
	}
	if _, err := os.Stat(bin); err != nil {
		b.Skip("build it first: task build")
	}
	benchProcess(b, bin)
}

func BenchmarkPythonProcess(b *testing.B) {
	script := os.Getenv("GITSTATUS_PY")
	if script == "" {
		b.Skip("set GITSTATUS_PY to the path of gitstatus.py")
	}
	if _, err := os.Stat(script); err != nil {
		b.Skipf("GITSTATUS_PY=%s: %v", script, err)
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		b.Skip("python3 not installed")
	}
	benchProcess(b, python, script)
}

func benchProcess(b *testing.B, name string, args ...string) {
	b.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		b.Skip("git not installed")
	}
	dir := benchRepo(b)

	for b.Loop() {
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		if _, err := cmd.Output(); err != nil {
			b.Fatalf("%s: %v", name, err)
		}
	}
}
