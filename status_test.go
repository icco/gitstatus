package main

import "testing"

func noDetached(t *testing.T) func() string {
	t.Helper()
	return func() string {
		t.Error("detached callback should not have been called")
		return ""
	}
}

func TestParseHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		line       string
		wantBranch string
		wantAhead  int
		wantBehind int
	}{
		{"no upstream", "## main", "main", 0, 0},
		{"upstream, in sync", "## main...origin/main", "main", 0, 0},
		{"ahead", "## main...origin/main [ahead 3]", "main", 3, 0},
		{"behind", "## main...origin/main [behind 2]", "main", 0, 2},
		{"diverged", "## main...origin/main [ahead 1, behind 4]", "main", 1, 4},
		{"behind listed first", "## m...origin/m [behind 4, ahead 1]", "m", 1, 4},
		{"upstream gone", "## main...origin/main [gone]", "main", 0, 0},
		{"initial commit", "## Initial commit on main", "main", 0, 0},
		{"no commits yet", "## No commits yet on main", "main", 0, 0},
		{"slashed branch", "## feat/thing...origin/feat/thing", "feat/thing", 0, 0},
		// The Python original raises ValueError unpacking this one.
		{"dots in branch name", "## we...ird...origin/x", "we", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parse([]byte(tt.line), noDetached(t))
			if !ok {
				t.Fatal("parse reported no branch header")
			}
			if got.Branch != tt.wantBranch {
				t.Errorf("branch = %q, want %q", got.Branch, tt.wantBranch)
			}
			if got.Ahead != tt.wantAhead {
				t.Errorf("ahead = %d, want %d", got.Ahead, tt.wantAhead)
			}
			if got.Behind != tt.wantBehind {
				t.Errorf("behind = %d, want %d", got.Behind, tt.wantBehind)
			}
		})
	}
}

func TestParseDetached(t *testing.T) {
	t.Parallel()
	got, ok := parse([]byte("## HEAD (no branch)"), func() string { return "v1.2.3+" })
	if !ok {
		t.Fatal("parse reported no branch header")
	}
	if got.Branch != "v1.2.3+" {
		t.Errorf("branch = %q, want the detached callback's value", got.Branch)
	}
}

func TestParseNoHeader(t *testing.T) {
	t.Parallel()
	// Status output with no `##` line: the caller must print nothing rather
	// than a Status whose empty branch would shift every field.
	if _, ok := parse([]byte(" M foo.go"), noDetached(t)); ok {
		t.Error("parse reported a branch header where there was none")
	}
}

func TestParseCounts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want Status
	}{
		{"clean", "", Status{Clean: 1}},
		{"untracked", "?? new.go", Status{Untracked: 1}},
		{"worktree modified", " M a.go", Status{Changed: 1}},
		{"staged", "M  a.go", Status{Staged: 1}},
		{"worktree deleted", " D a.go", Status{Deleted: 1}},
		{"staged delete", "D  a.go", Status{Staged: 1}},
		{"added", "A  a.go", Status{Staged: 1}},
		{"renamed", "R  a.go -> b.go", Status{Staged: 1}},
		// Quirks inherited from gitstatus.py: the Y tests are independent
		// ifs, so a staged-and-modified path lands in two buckets.
		{"staged and modified counts twice", "MM a.go", Status{Changed: 1, Staged: 1}},
		{"staged and deleted counts twice", "MD a.go", Status{Deleted: 1, Staged: 1}},
		{"unmerged both modified", "UU a.go", Status{Conflicts: 1}},
		{"unmerged deleted by them", "UD a.go", Status{Deleted: 1, Conflicts: 1}},
		// X is 'A', not 'U', so this reads as staged even though git calls it
		// unmerged. Faithful to the original.
		{"added by us reads as staged", "AU a.go", Status{Staged: 1}},
		{"mixed", "?? n.go\n M a.go\nM  b.go\n D c.go", Status{Untracked: 1, Changed: 1, Staged: 1, Deleted: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := "## main"
			if tt.body != "" {
				in += "\n" + tt.body
			}
			got, ok := parse([]byte(in), noDetached(t))
			if !ok {
				t.Fatal("parse reported no branch header")
			}
			want := tt.want
			want.Branch = "main"
			if got != want {
				t.Errorf("parse() = %+v\n          want %+v", got, want)
			}
		})
	}
}

func TestParseIgnoresBlankAndShortLines(t *testing.T) {
	t.Parallel()
	got, ok := parse([]byte("## main\n\n?? a.go\n"), noDetached(t))
	if !ok {
		t.Fatal("parse reported no branch header")
	}
	if got.Untracked != 1 || got.Clean != 0 {
		t.Errorf("parse() = %+v, want exactly one untracked file", got)
	}
}

func TestStatusStringFieldOrder(t *testing.T) {
	t.Parallel()
	s := Status{
		Branch: "main", Ahead: 1, Behind: 2, Staged: 3, Conflicts: 4,
		Changed: 5, Untracked: 6, Stashed: 7, Clean: 8, Deleted: 9,
	}
	const want = "main 1 2 3 4 5 6 7 8 9"
	if got := s.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
