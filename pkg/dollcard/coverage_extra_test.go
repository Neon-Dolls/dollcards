package dollcard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── ValidatePath branch coverage ───────────────────────────────────────────

// TestValidatePath_ArchiveNotZip hits the "archive:" problem branch (file is
// not a .dollcard ZIP).
func TestValidatePath_ArchiveNotZip(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.dollcard")
	if err := os.WriteFile(f, []byte("this is not a zip"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	ps := ValidatePath(f)
	if len(ps) != 1 {
		t.Fatalf("expected exactly one problem, got %v", ps)
	}
	if !strings.Contains(ps[0].Msg, "archive:") {
		t.Fatalf("expected an archive problem, got %v", ps[0].Msg)
	}
}

// TestValidatePath_UnreadableDir hits the "index:" problem branch by making
// the directory unreadable so IndexDir fails.
func TestValidatePath_UnreadableDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(dir, 0777); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.Chmod(dir, 0000); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	defer os.Chmod(dir, 0777) // restore so TempDir can clean up
	ps := ValidatePath(dir)
	if len(ps) != 1 {
		t.Fatalf("expected exactly one problem, got %v", ps)
	}
	if !strings.Contains(ps[0].Msg, "index:") {
		t.Fatalf("expected an index problem, got %v", ps[0].Msg)
	}
}

// ── parseHeader branch coverage ────────────────────────────────────────────

// TestParseHeader_InvalidJSON hits the "# not valid JSON" branch.
func TestParseHeader_InvalidJSON(t *testing.T) {
	_, ps := parseHeader([]byte("{not valid json"))
	if len(ps) != 1 {
		t.Fatalf("expected one problem, got %v", ps)
	}
	if !strings.Contains(ps[0].Msg, "not valid JSON") {
		t.Fatalf("unexpected problem: %v", ps[0].Msg)
	}
}

// TestParseHeader_UnsupportedVersion hits the version-mismatch branch.
func TestParseHeader_UnsupportedVersion(t *testing.T) {
	_, ps := parseHeader([]byte(`{"version":99}`))
	if len(ps) != 1 {
		t.Fatalf("expected one problem, got %v", ps)
	}
	if !strings.Contains(ps[0].Msg, "unsupported card version") {
		t.Fatalf("unexpected problem: %v", ps[0].Msg)
	}
}

// ── WriteArchive error branch ──────────────────────────────────────────────

// TestWriteArchive_OutputIsDir hits the "dollcard: write" branch when the
// destination path resolves to a directory rather than a writable file.
func TestWriteArchive_OutputIsDir(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out.dollcard")
	if err := os.MkdirAll(dst, 0777); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	files := map[string][]byte{"card.json": []byte(`{"version":1}`)}
	err := WriteArchive(files, dst)
	if err == nil {
		t.Fatalf("expected an error writing to a directory, got nil")
	}
	if !strings.Contains(err.Error(), "dollcard: write") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── Create error branches ──────────────────────────────────────────────────

// TestCreate_SubdirIsFile hits the MkdirAll failure branch in the folder loop.
func TestCreate_SubdirIsFile(t *testing.T) {
	dir := t.TempDir()
	// Pre-create one required subfolder as a FILE so MkdirAll fails there.
	if err := os.WriteFile(filepath.Join(dir, "owner"), []byte("file"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	err := Create(dir, "d-1", "Ruby")
	if err == nil {
		t.Fatalf("expected error creating a subdir over a file, got nil")
	}
	if !strings.Contains(err.Error(), "dollcard: create") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCreate_CardJsonIsDir hits the write card.json failure branch.
func TestCreate_CardJsonIsDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "card.json"), 0777); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	err := Create(dir, "d-1", "Ruby")
	if err == nil {
		t.Fatalf("expected error writing card.json, got nil")
	}
	if !strings.Contains(err.Error(), "write card.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCreate_IdentityJsonIsDir hits the write identity.json failure branch.
func TestCreate_IdentityJsonIsDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "identity.json"), 0777); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	err := Create(dir, "d-1", "Ruby")
	if err == nil {
		t.Fatalf("expected error writing identity.json, got nil")
	}
	if !strings.Contains(err.Error(), "write identity.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCreate_SoulMdIsDir hits the write soul.md failure branch.
func TestCreate_SoulMdIsDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "soul.md"), 0777); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	err := Create(dir, "d-1", "Ruby")
	if err == nil {
		t.Fatalf("expected error writing soul.md, got nil")
	}
	if !strings.Contains(err.Error(), "write soul.md") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCreate_OwnerMdIsDir hits the write owner/owner.md failure branch.
func TestCreate_OwnerMdIsDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "owner", "owner.md"), 0777); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	err := Create(dir, "d-1", "Ruby")
	if err == nil {
		t.Fatalf("expected error writing owner/owner.md, got nil")
	}
	if !strings.Contains(err.Error(), "write owner/owner.md") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── Compress error branches ────────────────────────────────────────────────

// TestCompress_IndexDirError hits the "index" problem branch (dir is a file).
func TestCompress_IndexDirError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("file"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	ps := Compress(src, filepath.Join(dir, "out.dollcard"))
	if len(ps) != 1 {
		t.Fatalf("expected one problem, got %v", ps)
	}
	if !strings.Contains(ps[0].Msg, "index") {
		t.Fatalf("expected index problem, got %v", ps[0].Msg)
	}
}

// TestCompress_WriteArchiveError hits the "compress:" problem branch (dst is a
// directory so WriteArchive fails after validation passes).
func TestCompress_WriteArchiveError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0777); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	// Write a fully valid card into src.
	if err := os.WriteFile(filepath.Join(src, "card.json"), []byte(`{"version":1}`), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "identity.json"), []byte(`{"doll_id":"d-1","canonical_name":"Ruby"}`), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "soul.md"), []byte("# Soul\n\nProse.\n"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(src, "owner"), 0777); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "owner", "owner.md"), []byte("# Owner\n\nProse.\n"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	// dst is a directory: WriteArchive will fail.
	dst := filepath.Join(dir, "out.dollcard")
	if err := os.MkdirAll(dst, 0777); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	ps := Compress(src, dst)
	if len(ps) != 1 {
		t.Fatalf("expected one problem, got %v", ps)
	}
	if !strings.Contains(ps[0].Msg, "compress:") {
		t.Fatalf("expected compress problem, got %v", ps[0].Msg)
	}
}

// ── Extract: existing empty destination ────────────────────────────────────

// TestExtract_EmptyDestination extracts into a pre-existing empty directory:
// exercises the isEmptyDir-true + Remove + Rename path.
func TestExtract_EmptyDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0777); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "card.json"), []byte(`{"version":1}`), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "identity.json"), []byte(`{"doll_id":"d-1","canonical_name":"Ruby"}`), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "soul.md"), []byte("# Soul\n\nProse.\n"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(src, "owner"), 0777); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "owner", "owner.md"), []byte("# Owner\n\nProse.\n"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	zipPath := filepath.Join(dir, "card.dollcard")
	if err := WriteArchive(map[string][]byte{
		"card.json":      []byte(`{"version":1}`),
		"identity.json":  []byte(`{"doll_id":"d-1","canonical_name":"Ruby"}`),
		"soul.md":        []byte("# Soul\n\nProse.\n"),
		"owner/owner.md": []byte("# Owner\n\nProse.\n"),
	}, zipPath); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0777); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	ps := Extract(zipPath, outDir)
	if len(ps) != 0 {
		t.Fatalf("expected success, got problems: %v", ps)
	}
	if vps := ValidatePath(outDir); len(vps) != 0 {
		t.Fatalf("extracted card did not validate: %v", vps)
	}
}

// ── isEmptyDir branch coverage ─────────────────────────────────────────────

// TestIsEmptyDir_Empty covers the true branch (existing, no entries).
func TestIsEmptyDir_Empty(t *testing.T) {
	dir := t.TempDir()
	if !isEmptyDir(dir) {
		t.Fatalf("expected empty dir to be reported empty")
	}
}

// TestIsEmptyDir_NonEmpty covers the false branch (existing, with entries).
func TestIsEmptyDir_NonEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x"), []byte("x"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if isEmptyDir(dir) {
		t.Fatalf("expected non-empty dir to be reported non-empty")
	}
}

// TestIsEmptyDir_Missing covers the ReadDir-error -> false branch.
func TestIsEmptyDir_Missing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	if isEmptyDir(dir) {
		t.Fatalf("expected missing dir to be reported not-empty (guard)")
	}
}

// ── collectDir error branch ────────────────────────────────────────────────

// TestCollectDir_UnreadableSubdir hits the recursive error propagation path:
// a readable root containing an unreadable subdirectory.
func TestCollectDir_UnreadableSubdir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "root.txt"), []byte("r"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0777); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.Chmod(sub, 0000); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	defer os.Chmod(sub, 0777) // restore for TempDir cleanup
	_, err := collectDir(root, "")
	if err == nil {
		t.Fatalf("expected error from unreadable subdir, got nil")
	}
}