// ZIP container reading and writing for Doll Cards.
//
// A version-1 .dollcard is an inspectable ZIP archive whose entries are the
// canonical relative paths of the card (card.json, identity.json, soul.md,
// owner/owner.md, and any optional areas). This module:
//
//   - reads an archive into the same canonical file map as paths.go produces
//     for a directory, applying the same path-safety rules (this rejects
//     ZIP-slip / path-traversal archives because unsafe names are errors);
//   - writes a file map back as a ZIP, preserving every entry losslessly.

package dollcard

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
)

// IndexArchive reads the .dollcard ZIP at zipPath and returns a map of
// canonical relative path → file bytes. Unsafe entry names (absolute paths,
// ".." traversal, drive prefixes) are rejected as errors: the whole archive
// is refused rather than partially trusted.
func IndexArchive(zipPath string) (map[string][]byte, error) {
	rc, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("dollcard: open %q: %w", zipPath, err)
	}
	defer rc.Close()
	return indexZipReader(&rc.Reader)
}

// indexZipReader extracts and validates every file entry from a ZIP reader.
func indexZipReader(zr *zip.Reader) (map[string][]byte, error) {
	files := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// Normalize and enforce safety on every entry name.
		name, serr := safeRelPath(f.Name)
		if serr != nil {
			return nil, fmt.Errorf("dollcard: unsafe archive path %q: %v", f.Name, serr)
		}
		rc, oerr := f.Open()
		if oerr != nil {
			return nil, fmt.Errorf("dollcard: read %q: %w", name, oerr)
		}
		data, rerr := io.ReadAll(rc)
		rc.Close()
		if rerr != nil {
			return nil, fmt.Errorf("dollcard: read %q: %w", name, rerr)
		}
		files[name] = data
	}
	return files, nil
}

// WriteArchive writes the file map to dstPath as a .dollcard ZIP. Entries are
// written in sorted order for deterministic output. Every entry is preserved
// losslessly. The parent directory of dstPath must already exist.
func WriteArchive(files map[string][]byte, dstPath string) error {
	// Buffer the archive in memory, then write it out in one shot so that a
	// mid-write failure cannot leave a truncated-but-valid-looking card.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	names := make([]string, len(files))
	i := 0
	for name := range files {
		names[i] = name
		i++
	}
	sort.Strings(names)

	for _, name := range names {
		w, cerr := zw.Create(name)
		if cerr != nil {
			return fmt.Errorf("dollcard: create entry %q: %w", name, cerr)
		}
		if _, werr := w.Write(files[name]); werr != nil {
			return fmt.Errorf("dollcard: write entry %q: %w", name, werr)
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("dollcard: finalize archive: %w", err)
	}

	if err := os.WriteFile(dstPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("dollcard: write %q: %w", dstPath, err)
	}
	return nil
}
