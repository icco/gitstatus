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

// BenchmarkSpikeGoGitStatus is the proposed replacement. No process spawn,
// but go-git walks the worktree and hashes file contents itself.
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
