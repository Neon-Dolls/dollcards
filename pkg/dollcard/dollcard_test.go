// Tests for the dollcard package: model, validation, container I/O, and
// high-level operations. All logic is package-internal (unexported helpers are
// exercised through the same file map the directory/archive indexers produce,
// matching how the CLI drives the library).

package dollcard

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Fixtures ────────────────────────────────────────────────────────────────

// validCard returns a canonical, fully-valid v1 card file map.
func validCard() map[string][]byte {
	return map[string][]byte{
		"card.json":      []byte(`{"version":1}`),
		"identity.json":  []byte(`{"doll_id":"d-1","canonical_name":"Ruby"}`),
		"soul.md":        []byte("# Soul\n\nSome prose.\n"),
		"owner/owner.md": []byte("# Owner\n\nSome prose.\n"),
	}
}

// requireName writes the test file map into a fresh temp dir and returns it.
func requireName(t *testing.T, dir string, files map[string][]byte) {
	t.Helper()
	for name, data := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0777); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(p, data, 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func hasProblem(ps []problem, path, substr string) bool {
	for _, p := range ps {
		if p.Path == path && strings.Contains(p.Msg, substr) {
			return true
		}
	}
	return false
}

// ── Validate: happy path ────────────────────────────────────────────────────

func TestValidate_ValidCard(t *testing.T) {
	if ps := Validate(validCard()); len(ps) != 0 {
		t.Fatalf("valid card produced problems: %v", ps)
	}
}

func TestValidate_MissingRequiredFiles(t *testing.T) {
	got := Validate(map[string][]byte{})
	want := len(requiredFiles)
	if len(got) != want {
		t.Fatalf("got %d problems, want %d: %v", len(got), want, got)
	}
	for _, name := range requiredFiles {
		if !hasProblem(got, name, "missing required file") {
			t.Errorf("expected missing-file problem for %q", name)
		}
	}
}

// ── Validate: card.json version ──

func TestValidate_CardVersion(t *testing.T) {
	cases := []struct {
		name string
		hdr  string
		ok   bool
	}{
		{"version 0", `{"version":0}`, false},
		{"version 2 unsupported", `{"version":2}`, false},
		{"not json", `not-json`, false},
		{"missing version", `{}`, false},
		{"valid v1", `{"version":1}`, true},
	}
	for _, c := range cases {
		files := validCard()
		files["card.json"] = []byte(c.hdr)
		ps := Validate(files)
		if c.ok && len(ps) != 0 {
			t.Errorf("%s: expected valid, got %v", c.name, ps)
		}
		if !c.ok && !hasProblem(ps, "card.json", "card version") && !hasProblem(ps, "card.json", "not valid JSON") {
			t.Errorf("%s: expected card.json problem, got %v", c.name, ps)
		}
	}
}

// ── Validate: identity ──

func TestValidate_Identity(t *testing.T) {
	files := validCard()
	files["identity.json"] = []byte(`{"doll_id":"","canonical_name":""}`)
	ps := Validate(files)
	if !hasProblem(ps, "identity.json", "missing doll_id") {
		t.Error("expected missing doll_id problem")
	}
	if !hasProblem(ps, "identity.json", "missing canonical_name") {
		t.Error("expected missing canonical_name problem")
	}

	files["identity.json"] = []byte(`not-json`)
	ps = Validate(files)
	if !hasProblem(ps, "identity.json", "not valid JSON") {
		t.Errorf("expected invalid-JSON problem, got %v", ps)
	}
}

// ── Validate: empty prose ──

func TestValidate_EmptyProse(t *testing.T) {
	files := validCard()
	files["soul.md"] = []byte("   \n  ")
	files["owner/owner.md"] = []byte("")
	ps := Validate(files)
	if !hasProblem(ps, "soul.md", "must not be empty") {
		t.Error("expected empty soul problem")
	}
	if !hasProblem(ps, "owner/owner.md", "must not be empty") {
		t.Error("expected empty owner problem")
	}
}

// ── Validate: optional structured JSON areas ──

func TestValidate_OptionalJSON(t *testing.T) {
	files := validCard()
	files["self/self.json"] = []byte(`{"display_name":"Ruby"}`)
	files["drives/drives.json"] = []byte(`{"items":[]}`)
	files["goals/goals.json"] = []byte(`[`) // truncated JSON
	if ps := Validate(files); !hasProblem(ps, "goals/goals.json", "malformed JSON") {
		t.Errorf("expected malformed JSON on goals, got %v", ps)
	}
}

// ── Validate: memory JSONL ──

func TestValidate_MemoryJSONL(t *testing.T) {
	files := validCard()
	files["memories/all.jsonl"] = []byte("{\"id\":1}\n\n{\"id\":2}\n")
	if ps := Validate(files); len(ps) != 0 {
		t.Fatalf("clean JSONL should validate, got %v", ps)
	}
	files["memories/all.jsonl"] = []byte("{\"id\":1}\nbroken\n")
	if ps := Validate(files); !hasProblem(ps, "memories/all.jsonl", "malformed memory record") {
		t.Errorf("expected malformed memory record, got %v", ps)
	}
}

// ── Validate: skills ──

func TestValidate_Skills(t *testing.T) {
	files := validCard()
	files["skills/brew/SKILL.md"] = []byte("# brew\n")
	if ps := Validate(files); len(ps) != 0 {
		t.Fatalf("complete skill should validate, got %v", ps)
	}
	// A skill directory without its SKILL.md is a partial payload.
	files["skills/brew/recipe.txt"] = []byte("x")
	delete(files, "skills/brew/SKILL.md")
	if ps := Validate(files); !hasProblem(ps, "skills/brew", "SKILL.md") {
		t.Errorf("expected missing SKILL.md, got %v", ps)
	}
}

// ── safeRelPath ──

func TestSafeRelPath(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"card.json", "card.json", false},
		{"owner/owner.md", "owner/owner.md", false},
		{"a/b/c.txt", "a/b/c.txt", false},
		{"a\\b\\c.txt", "a/b/c.txt", false}, // backslash normalized
		{"./owner/owner.md", "owner/owner.md", false},
		{"//x", "", true},         // absolute
		{"/etc/passwd", "", true}, // absolute
		{"a/../../etc", "", true}, // traversal
		{"C:\\x", "", true},       // drive prefix
		{"..", "", true},          // traversal
		{"a/./b", "a/b", false},
	}
	for _, c := range cases {
		got, err := safeRelPath(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("safeRelPath(%q): expected error, got %q", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("safeRelPath(%q): unexpected error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("safeRelPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ── IndexDir / WriteArchive round-trip ──

func TestIndexDir_And_ArchiveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	requireName(t, dir, validCard())
	files, err := IndexDir(dir)
	if err != nil {
		t.Fatalf("IndexDir: %v", err)
	}
	if len(files) != len(validCard()) {
		t.Fatalf("indexed %d files, want %d", len(files), len(validCard()))
	}

	// Write to archive, then read back.
	zpath := filepath.Join(dir, "card.dollcard")
	if err := WriteArchive(files, zpath); err != nil {
		t.Fatalf("WriteArchive: %v", err)
	}
	back, err := IndexArchive(zpath)
	if err != nil {
		t.Fatalf("IndexArchive: %v", err)
	}
	if len(back) != len(files) {
		t.Fatalf("round-trip changed file count: %d -> %d", len(files), len(back))
	}
	for name, data := range files {
		if !bytes.Equal(back[name], data) {
			t.Errorf("round-trip changed %q", name)
		}
	}
}

func TestIndexArchive_RejectsUnsafePath(t *testing.T) {
	dir := t.TempDir()
	zpath := filepath.Join(dir, "evil.dollcard")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("../escape.md")
	w.Write([]byte("x"))
	zw.Close()
	if err := os.WriteFile(zpath, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := IndexArchive(zpath); err == nil {
		t.Fatal("expected unsafe-path rejection, got nil")
	}
}

func TestIndexArchive_NotAZip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "junk.dollcard")
	os.WriteFile(p, []byte("definitely not a zip"), 0644)
	if _, err := IndexArchive(p); err == nil {
		t.Fatal("expected error opening non-zip, got nil")
	}
}

// ── Create / GenerateID ──

func TestCreate_Valid(t *testing.T) {
	dir := t.TempDir()
	card := filepath.Join(dir, "new")
	if err := Create(card, "id-1", "Ruby"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if ps := ValidatePath(card); len(ps) != 0 {
		t.Fatalf("created card is invalid: %v", ps)
	}
}

func TestCreate_RequiresName(t *testing.T) {
	if err := Create(t.TempDir(), "id-1", ""); err == nil {
		t.Fatal("expected error when canonical name empty")
	}
}

func TestGenerateID_Shape(t *testing.T) {
	id, err := GenerateID()
	if err != nil {
		t.Fatalf("GenerateID() returned error: %v", err)
	}
	// UUIDv4: 8-4-4-4-12, version digit 4 in third group.
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Fatalf("GenerateID() = %q, not UUID-shaped (got %d parts)", id, len(parts))
	}
	if len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 || len(parts[3]) != 4 || len(parts[4]) != 12 {
		t.Fatalf("GenerateID() = %q, wrong segment lengths", id)
	}
	if parts[2][0] != '4' {
		t.Errorf("GenerateID() = %q, version nibble = %c, want 4", id, parts[2][0])
	}
}

// ── Compress / Extract ──

func TestCompress_Then_Extract(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	requireName(t, src, validCard())

	out := filepath.Join(dir, "c.dollcard")
	if ps := Compress(src, out); len(ps) != 0 {
		t.Fatalf("Compress: %v", ps)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("archive not written: %v", err)
	}

	dest := filepath.Join(dir, "dest")
	if ps := Extract(out, dest); len(ps) != 0 {
		t.Fatalf("Extract: %v", ps)
	}
	back, err := IndexDir(dest)
	if err != nil {
		t.Fatalf("reindex extracted: %v", err)
	}
	if len(back) != len(validCard()) {
		t.Errorf("extracted %d files, want %d", len(back), len(validCard()))
	}
}

func TestCompress_RejectsInvalidCard(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bad")
	// No required files -> invalid.
	requireName(t, src, map[string][]byte{"stray.txt": []byte("x")})
	out := filepath.Join(dir, "c.dollcard")
	if ps := Compress(src, out); len(ps) == 0 {
		t.Fatal("expected validation failure, got none")
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("invalid card must not be compressed")
	}
}

func TestExtract_RejectsInvalidArchive(t *testing.T) {
	dir := t.TempDir()
	// A zip that is not a valid card.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("stray.txt")
	w.Write([]byte("x"))
	zw.Close()
	zp := filepath.Join(dir, "bad.dollcard")
	os.WriteFile(zp, buf.Bytes(), 0644)

	dest := filepath.Join(dir, "dest")
	if ps := Extract(zp, dest); len(ps) == 0 {
		t.Fatal("expected rejection of invalid archive, got none")
	}
	if _, err := os.Stat(dest); err == nil {
		t.Fatal("destination must not be created for invalid archive")
	}
}

func TestExtract_NonEmptyDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	requireName(t, src, validCard())
	out := filepath.Join(dir, "c.dollcard")
	if ps := Compress(src, out); len(ps) != 0 {
		t.Fatalf("compress: %v", ps)
	}
	dest := filepath.Join(dir, "occupied")
	os.MkdirAll(dest, 0777)
	os.WriteFile(filepath.Join(dest, "x.txt"), []byte("y"), 0644)
	if ps := Extract(out, dest); len(ps) == 0 {
		t.Fatal("expected error for non-empty destination, got none")
	}
	// The pre-existing file must be untouched.
	if data, _ := os.ReadFile(filepath.Join(dest, "x.txt")); string(data) != "y" {
		t.Error("existing destination content was clobbered")
	}
}

// ── ValidatePath / DescribeProblems (public API) ──

func TestValidatePath_DirAndArchive(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	requireName(t, src, validCard())
	if ps := ValidatePath(src); len(ps) != 0 {
		t.Fatalf("ValidatePath(dir): %v", ps)
	}
	out := filepath.Join(dir, "c.dollcard")
	if ps := Compress(src, out); len(ps) != 0 {
		t.Fatalf("compress: %v", ps)
	}
	if ps := ValidatePath(out); len(ps) != 0 {
		t.Fatalf("ValidatePath(archive): %v", ps)
	}
}

func TestValidatePath_Missing(t *testing.T) {
	ps := ValidatePath(filepath.Join(t.TempDir(), "nope"))
	if !hasProblem(ps, "", "path does not exist") {
		t.Errorf("expected does-not-exist, got %v", ps)
	}
}

func TestDescribeProblems(t *testing.T) {
	ps := []problem{Problem("", "whole card"), Problem("soul.md", "empty")}
	got := DescribeProblems(ps)
	if len(got) != 2 {
		t.Fatalf("DescribeProblems -> %v", got)
	}
	if !strings.HasPrefix(got[0], "(card):") {
		t.Errorf("empty path should read (card): got %q", got[0])
	}
	if !strings.HasPrefix(got[1], "soul.md:") {
		t.Errorf("path should be printed: got %q", got[1])
	}
}
