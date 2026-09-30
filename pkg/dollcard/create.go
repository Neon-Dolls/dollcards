// Creating a new editable Doll Card directory.
//
// `dollcard create <dir>` builds a real directory the user can edit and then
// compress into a .dollcard archive:
//
//   create → (edit) → validate → compress
//
// The produced directory immediately passes `dollcard validate` as a minimal
// Doll Card: it contains the required files with valid index, identity, Soul
// and Owner content, plus the canonical empty area directories so the folder
// structure is discoverable.

package dollcard

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// defaultSoul and defaultOwner supply the minimal non-empty prose required by
// version 1 validation. They must not be empty after trimming.
const defaultSoul = "# A Doll\n\nThis is where her Soul is defined: the prose that describes who she is, fundamentally.\n"
const defaultOwner = "# Owner\n\nWrite about the owner of this Doll here.\n"

// Create builds a new, minimal, valid Doll Card directory at dir.
// canonName is the canonical_name written to identity.json; dollID is the
// doll_id (may be "" to leave for the caller; validation requires it).
func Create(dir, dollID, canonName string) error {
	if canonName == "" {
		return fmt.Errorf("dollcard: canonical name is required")
	}
	folders := []string{"", "owner", "self", "avatar", "relationships", "preferences", "memories", "secrets", "skills", "drives", "goals", "extensions"}
	for _, sub := range folders {
		path := dir
		if sub != "" {
			path = filepath.Join(dir, sub)
		}
		if err := os.MkdirAll(path, 0777); err != nil {
			return fmt.Errorf("dollcard: create %q: %w", path, err)
		}
	}

	// card.json — the version index.
	header, err := json.Marshal(cardHeader{Version: currentCardVersion})
	if err != nil {
		return fmt.Errorf("dollcard: marshal card.json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "card.json"), header, 0644); err != nil {
		return fmt.Errorf("dollcard: write card.json: %w", err)
	}

	// identity.json — structured identity.
	id, err := json.Marshal(identity{
		DollID:        dollID,
		CanonicalName: canonName,
	})
	if err != nil {
		return fmt.Errorf("dollcard: marshal identity.json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "identity.json"), id, 0644); err != nil {
		return fmt.Errorf("dollcard: write identity.json: %w", err)
	}

	// soul.md and owner/owner.md.
	if err := os.WriteFile(filepath.Join(dir, "soul.md"), []byte(defaultSoul), 0644); err != nil {
		return fmt.Errorf("dollcard: write soul.md: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "owner", "owner.md"), []byte(defaultOwner), 0644); err != nil {
		return fmt.Errorf("dollcard: write owner/owner.md: %w", err)
	}
	return nil
}

// GenerateID returns a fresh random doll_id (UUIDv4-shaped). It is a helper
// so the CLI can create a card with a stable identifier when none is given.
func GenerateID() string {
	data := make([]byte, 16)
	rand.Read(data)
	data[6] = byte(int(data[6])&0x0f | 0x40) // version 4
	data[8] = byte(int(data[8])&0x3f | 0x80) // variant 1
	hex := "0123456789abcdef"
	var sb strings.Builder
	for i := range data {
		b := int(data[i])
		sb.WriteByte(byte(hex[b>>4]))
		sb.WriteByte(byte(hex[b&15]))
		if i == 3 || i == 5 || i == 7 || i == 9 {
			sb.WriteByte(byte('-'))
		}
	}
	return sb.String()
}
