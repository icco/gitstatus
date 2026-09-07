package main

// Spike: is go-git's Worktree.Status() actually faster than forking
// `git status --porcelain --branch`? Everything else in a port is small
// compared to this one call, so it decides the question on its own.
//
// Run against a big repository too, not just this one:
//
//	SPIKE_REPO=~/Projects/some-large-repo go test -bench Spike -run '^$' ./...

import (
	"os"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	gogitv6 "github.com/go-git/go-git/v6"
)

func spikeDir(b *testing.B) string {
	b.Helper()
	if d := os.Getenv("SPIKE_REPO"); d != "" {
		return d
	}
	wd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	return wd
}

// BenchmarkSpikeForkGitStatus is the cost we pay today: one process spawn
// plus git's own highly optimised status, which leans on the index stat cache.
func BenchmarkSpikeForkGitStatus(b *testing.B) {
	b.Chdir(spikeDir(b))
	for b.Loop() {
		if _, err := git("status", "--porcelain", "--branch"); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSpikeGoGitStatus is the proposed replacement: no process spawn.
//
// Profiling this on a 4317-file worktree puts 94.8% of the time in
// syscall.rawsyscalln -- ReadDir 49.7% cumulative, Lstat 29.0%, Open 28.4%,
// and go-billy's resolveFollowedPath (per-path symlink resolution) 20.0%.
// Nothing measurable is spent hashing. The cost is filesystem syscalls
// through go-billy's virtual filesystem, not content hashing.
func BenchmarkSpikeGoGitStatus(b *testing.B) {
	dir := spikeDir(b)
	for b.Loop() {
		repo, err := gogit.PlainOpenWithOptions(dir, &gogit.PlainOpenOptions{DetectDotGit: true})
		if err != nil {
			b.Fatal(err)
		}
		wt, err := repo.Worktree()
		if err != nil {
			b.Fatal(err)
		}
		if _, err := wt.Status(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSpikeGoGitStatusReuseOpen separates opening the repository from
// computing status, in case PlainOpen dominates. A real command pays both, but
// this shows where the time goes.
func BenchmarkSpikeGoGitStatusReuseOpen(b *testing.B) {
	dir := spikeDir(b)
	repo, err := gogit.PlainOpenWithOptions(dir, &gogit.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		b.Fatal(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		if _, err := wt.Status(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSpikeGoGitV6Status tests whether v6 closes the gap. Its changelog
// claims "Improve Status() speed with new index.ModTime check" and a
// "stat-based index cache" -- but the v5 profile shows the cost is directory
// walking and per-path Lstat through go-billy, not index hashing, so an index
// cache may not touch the bottleneck.
func BenchmarkSpikeGoGitV6Status(b *testing.B) {
	dir := spikeDir(b)
	for b.Loop() {
		repo, err := gogitv6.PlainOpenWithOptions(dir, &gogitv6.PlainOpenOptions{DetectDotGit: true})
		if err != nil {
			b.Fatal(err)
		}
		wt, err := repo.Worktree()
		if err != nil {
			b.Fatal(err)
		}
		if _, err := wt.Status(); err != nil {
			b.Fatal(err)
		}
	}
}
