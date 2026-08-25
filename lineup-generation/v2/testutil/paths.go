// Package testutil resolves paths to files inside this module (fixtures, schedules)
// relative to the source tree, so tests and mock loaders work from any checkout
// location instead of depending on a hardcoded absolute path.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// scheduleCandidates lists the bundled schedule files newest season first. Tests use
// the first one that exists so the suite keeps working while a new season's file is
// being added and after the old one is removed.
var scheduleCandidates = []string{"schedule26-27.json", "schedule25-26.json"}

// Root returns the absolute path of the v2 module root (the directory containing go.mod),
// resolved from this source file's location at compile time.
func Root() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("testutil: unable to determine source file location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

// RepoPath joins the given path segments onto the module root.
func RepoPath(parts ...string) string {
	return filepath.Join(append([]string{Root()}, parts...)...)
}

// SchedulePath returns the path of the newest bundled schedule file under static/.
func SchedulePath() (string, error) {
	for _, name := range scheduleCandidates {
		path := RepoPath("static", name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("testutil: none of %v found under %s", scheduleCandidates, RepoPath("static"))
}
