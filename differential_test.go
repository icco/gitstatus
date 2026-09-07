package main

import (
	"os"
	"os/exec"
	"testing"
)

// TestRenderMatchesPython compares this implementation byte-for-byte against
// the gitstatus.py it replaces, over every scenario in states().
//
// Opt in by pointing GITSTATUS_PY at the script, e.g.
//
//	GITSTATUS_PY=~/.oh-my-zsh/plugins/git-prompt/gitstatus.py go test -run Python
//
// It is skipped by default so the suite needs neither python3 nor a copy of a
// script this repo does not own.
func TestRenderMatchesPython(t *testing.T) {
	script := os.Getenv("GITSTATUS_PY")
	if script == "" {
		t.Skip("set GITSTATUS_PY to the path of gitstatus.py to run the differential test")
	}
	if _, err := os.Stat(script); err != nil {
		t.Skipf("GITSTATUS_PY=%s: %v", script, err)
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not installed")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	for _, st := range states() {
		t.Run(st.name, func(t *testing.T) {
			dir := t.TempDir()
			st.setup(t, dir)
			runIn := st.dirFor(dir)
			t.Chdir(runIn)

			// Errors are deliberately ignored: where the script raises, it
			// prints nothing, and empty output is exactly what we compare
			// against because that is all the prompt ever sees.
			cmd := exec.Command(python, script)
			cmd.Dir = runIn
			pyOut, _ := cmd.Output()

			if got, want := render(), string(pyOut); got != want {
				t.Errorf("go  = %q\npy  = %q", got, want)
			}
		})
	}
}
