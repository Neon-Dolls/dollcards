// Public entrypoints for the dollcard package.
//
// These are the functions the CLI (and future tooling) call. They route a
// path to the appropriate directory/archive path, keeping callers agnostic to
// whether they are validating, compressing, or extracting a directory or a
// .dollcard file.

package dollcard

import (
	"fmt"
	"os"
)

// ValidatePath validates whatever path refers to: a directory (unpacked
// Doll Card) or a .dollcard archive. It returns the list of problems (empty
// list = valid).
func ValidatePath(path string) []problem {
	if _, err := os.Stat(path); err != nil {
		return []problem{Problem("", "path does not exist: "+path)}
	}
	if info, _ := os.Lstat(path); info != nil && info.IsDir() {
		files, err := IndexDir(path)
		if err != nil {
			return []problem{Problem("", "index: "+err.Error())}
		}
		return Validate(files)
	}
	files, err := IndexArchive(path)
	if err != nil {
		return []problem{Problem("", "archive: "+err.Error())}
	}
	return Validate(files)
}

// DescribeProblems formats the problem list into human-readable lines for
// printing to a terminal.
func DescribeProblems(ps []problem) []string {
	var out []string
	for _, p := range ps {
		where := p.Path
		if where == "" {
			where = "(card)"
		}
		out = append(out, fmt.Sprintf("%s: %s", where, p.Msg))
	}
	return out
}
