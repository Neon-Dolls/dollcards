// Compress and extract for Doll Cards.
//
// compress: validate a directory, then write it as a canonical .dollcard ZIP.
// extract:  validate an archive, then safely unpack it into a directory.
//
// Both are validation-first: a card that does not pass Validate is neither
// compressed nor extracted. Extraction is additionally hardened against
// ZIP-slip/path traversal (rejected in IndexArchive) and writes into a
// temporary staging directory that is only moved into place once the
// extracted result has re-validated — so a failed extraction cannot leave
// behind something that looks like a successfully extracted Doll Card.

package dollcard

import (
	"os"
	"path/filepath"
	"sort"
)

// Compress validates the Doll Card directory at dir and, if valid, writes it
// losslessly to dstPath as a .dollcard ZIP. dir is indexed (never followed for
// symlinks), so nothing outside the Doll Card tree can ever be included.
//
// Returns a non-empty problem list (and writes nothing) if validation fails.
func Compress(dir, dstPath string) []problem {
	files, err := IndexDir(dir)
	if err != nil {
		return []problem{Problem("", "index "+dir+": "+err.Error())}
	}
	ps := Validate(files)
	if ps != nil {
		return ps
	}
	if err := WriteArchive(files, dstPath); err != nil {
		return []problem{Problem("", "compress: "+err.Error())}
	}
	return nil
}

// Extract validates the .dollcard archive at zipPath and, if valid and safe,
// unpacks it into the directory at dir (created if needed). All archive entry
// names are safety-checked (see IndexArchive), so path traversal is rejected.
//
// A failed extraction leaves nothing that looks like a successfully extracted
// card: files are written to a temporary staging directory that is renamed
// into place only after the result re-validates.
func Extract(zipPath, dir string) []problem {
	files, err := IndexArchive(zipPath)
	if err != nil {
		return []problem{Problem("", "extract: "+err.Error())}
	}

	ps := Validate(files)
	if ps != nil {
		// Refuse to touch the destination: extracting a broken card would
		// create a directory that looks like a successful extraction.
		return ps
	}

	// Stage into a temp sibling dir, then move into place on success.
	parent := filepath.Dir(filepath.Clean(dir))
	staging, terr := os.MkdirTemp(parent, "dollcard-extract-")
	if terr != nil {
		return []problem{Problem("", "extract: staging: "+terr.Error())}
	}
	defer os.RemoveAll(staging)

	names := make([]string, len(files))
	i := 0
	for name := range files {
		names[i] = name
		i++
	}
	sort.Strings(names)

	for _, name := range names {
		target := filepath.Join(staging, name)
		if err := os.MkdirAll(filepath.Dir(target), 0777); err != nil {
			return []problem{Problem(name, "extract: mkdir: "+err.Error())}
		}
		if err := os.WriteFile(target, files[name], 0644); err != nil {
			return []problem{Problem(name, "extract: write: "+err.Error())}
		}
	}

	// Re-validate what we actually extracted; only then move it into place.
	files2, err2 := IndexDir(staging)
	if err2 != nil {
		return []problem{Problem("", "extract: revalidate: "+err2.Error())}
	}
	ps2 := Validate(files2)
	if ps2 != nil {
		return ps2
	}

	// Move staging into the final location. If dir already exists, require it
	// to be an empty directory so we never clobber existing content.
	if _, statErr := os.Stat(dir); statErr == nil {
		if !isEmptyDir(dir) {
			return []problem{Problem("", "extract: destination exists and is not empty: "+dir)}
		}
		if err := os.Remove(dir); err != nil {
			return []problem{Problem("", "extract: remove existing destination: "+err.Error())}
		}
	}
	if err := os.Rename(staging, dir); err != nil {
		return []problem{Problem("", "extract: move into place: "+err.Error())}
	}
	return nil
}

// isEmptyDir reports whether dir exists and contains no entries.
func isEmptyDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	return len(entries) == 0
}
