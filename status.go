package main

import (
	"strconv"
	"strings"
)

// Status is the ten-field summary a shell prompt consumes.
type Status struct {
	Branch    string
	Ahead     int
	Behind    int
	Staged    int
	Conflicts int
	Changed   int
	Untracked int
	Stashed   int
	Clean     int
	Deleted   int
}

// String renders the fields in the order the git-prompt plugin indexes them.
// No trailing newline: the plugin captures this in a $(...) and splits on
// spaces, and a newline would land in the last field.
func (s Status) String() string {
	return strings.Join([]string{
		s.Branch,
		strconv.Itoa(s.Ahead),
		strconv.Itoa(s.Behind),
		strconv.Itoa(s.Staged),
		strconv.Itoa(s.Conflicts),
		strconv.Itoa(s.Changed),
		strconv.Itoa(s.Untracked),
		strconv.Itoa(s.Stashed),
		strconv.Itoa(s.Clean),
		strconv.Itoa(s.Deleted),
	}, " ")
}

// parse turns `git status --porcelain --branch` output into a Status. detached
// is consulted only for a detached HEAD, so tests need not run git.
//
// The second return is false when the output carried no `##` header. The Python
// original raises NameError there and prints nothing; callers should likewise
// emit nothing rather than a Status with an empty branch, which would render as
// a leading space and shift every field the plugin reads.
//
// Stashed is not set here. It comes from a separate file read, and Clean
// deliberately ignores it, matching the original.
func parse(out []byte, detached func() string) (Status, bool) {
	var s Status
	haveBranch := false

	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if len(line) < 2 {
			continue
		}
		x, y := line[0], line[1]
		rest := ""
		if len(line) > 2 {
			rest = line[2:]
		}

		switch {
		case x == '#' && y == '#':
			s.Branch, s.Ahead, s.Behind = parseHeader(rest, detached)
			haveBranch = true
		case x == '?' && y == '?':
			s.Untracked++
		default:
			// Mirrors gitstatus.py deliberately: the Y tests are independent
			// ifs rather than a chain, so "MM" counts as both changed and
			// staged, and "MD" as both deleted and staged. Only the X tests
			// are exclusive. Reproducing this keeps the prompt's numbers
			// identical to the script this replaces.
			if y == 'M' {
				s.Changed++
			}
			if y == 'D' {
				s.Deleted++
			}
			if x == 'U' {
				s.Conflicts++
			} else if x != ' ' {
				s.Staged++
			}
		}
	}

	if s.Changed == 0 && s.Deleted == 0 && s.Staged == 0 && s.Conflicts == 0 && s.Untracked == 0 {
		s.Clean = 1
	}
	return s, haveBranch
}

// parseHeader reads the `## ...` line: branch name plus ahead/behind counts.
func parseHeader(rest string, detached func() string) (branch string, ahead, behind int) {
	switch {
	case strings.Contains(rest, "Initial commit on"), strings.Contains(rest, "No commits yet on"):
		f := strings.Split(rest, " ")
		return f[len(f)-1], 0, 0
	case strings.Contains(rest, "no branch"):
		return detached(), 0, 0
	}

	t := strings.TrimSpace(rest)
	// SplitN with a limit of 2: a branch whose name contains "..." makes the
	// original's two-value unpack raise ValueError. Here the extra dots stay
	// with the upstream half, which is the harmless reading.
	parts := strings.SplitN(t, "...", 2)
	if len(parts) == 1 {
		return t, 0, 0
	}
	branch = parts[0]

	fields := strings.Split(parts[1], " ")
	if len(fields) == 1 {
		// Upstream is set but has not diverged.
		return branch, 0, 0
	}

	div := strings.Join(fields[1:], " ")
	div = strings.TrimLeft(div, "[")
	div = strings.TrimRight(div, "]")
	for _, d := range strings.Split(div, ", ") {
		// Substring tests, not prefix tests, to match the original. Anything
		// else git puts here (such as "gone") matches neither and is ignored.
		switch {
		case strings.Contains(d, "ahead"):
			ahead = countAfter(d, "ahead ")
		case strings.Contains(d, "behind"):
			behind = countAfter(d, "behind ")
		}
	}
	return branch, ahead, behind
}

// countAfter reads the integer following a fixed-width label.
func countAfter(s, label string) int {
	if len(s) < len(label) {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(s[len(label):]))
	if err != nil {
		return 0
	}
	return n
}
