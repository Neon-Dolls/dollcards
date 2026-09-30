// Safe, canonical relative-path handling for Doll Card entries.
//
// Both directory walking and archive reading produce, for every Doll Card
// entry, a canonical slash-separated relative path. Paths are validated here
// so that:
//
//   - backslash separators are normalized to "/";
//   - absolute paths, drive-letter prefixes, and ".." traversal are rejected;
//   - every accepted path stays within the Doll Card tree.
//
// This is what makes extraction safe against ZIP-slip / path-traversal
// archives: if a name cannot be canonicalized safely, the whole archive is
// rejected rather than partially extracted.

package dollcard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// safeRelPath normalizes an entry name (as found in an archive or as built
// by a directory walk) into a canonical relative path with "/" separators.
// It returns an error for any name that would escape the card tree, is
// absolute, or is otherwise unsafe.
func safeRelPath(name string) (string, error) {
	// Normalize backslash separators to forward slashes so that a
	// Windows-produced archive cannot smuggle traversal via "\".
	n := strings.ReplaceAll(name, "\\", "/")

	if strings.HasPrefix(n, "/") {
		return "", fmt.Errorf("absolute path")
	}

	parts := strings.Split(n, "/")
	var out []string
	for _, p := range parts {
		switch p {
		case "", ".":
			// Skip empty and current-dir segments.
		case "..":
			return "", fmt.Errorf("path traversal")
		default:
			// Reject drive-letter / protocol-style prefixes ("C:").
			if strings.Index(p, ":") >= 0 {
				return "", fmt.Errorf("unsafe path component %q", p)
			}
			out = append(out, p)
		}
	}
	if out == nil {
		return "", fmt.Errorf("empty path")
	}
	return strings.Join(out, "/"), nil
}

// IndexDir walks the tree rooted at dir and returns a map of canonical
// relative path → file bytes. Only regular files are included; symlinks and
// other special files are skipped, and the walk does not follow symlinks, so
// it cannot escape the Doll Card tree.
func IndexDir(dir string) (map[string][]byte, error) {
	return collectDir(dir, "")
}

// collectDir recursively reads dir and returns a map of canonical relative
// path → file bytes under the current relative prefix.
func collectDir(dir, rel string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return files, fmt.Errorf("read directory %q: %w", dir, err)
	}
	for _, e := range entries {
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		// Skip symlinks (never follow them) and non-dir/non-file entries.
		if e.Type()&os.ModeSymlink != 0 {
			continue
		}
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		safe, serr := safeRelPath(childRel)
		if serr != nil {
			return files, fmt.Errorf("unsafe entry %q: %v", childRel, serr)
		}
		full := filepath.Join(dir, name)
		if e.IsDir() {
			sub, s2 := collectDir(full, safe)
			if s2 != nil {
				return files, s2
			}
			for k, v := range sub {
				files[k] = v
			}
			continue
		}
		data, rerr := os.ReadFile(full)
		if rerr != nil {
			return files, fmt.Errorf("read %q: %w", safe, rerr)
		}
		files[safe] = data
	}
	return files, nil
}
