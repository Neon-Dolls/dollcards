// Doll Card validation.
//
// Validate produces a list of human-readable problems describing everything
// wrong with a Doll Card. An empty list means the card is valid; callers
// (the CLI, compress, extract) treat a non-empty list as failure.
//
// Validation is intentionally independent of Core: it reasons only about the
// canonical file map produced by paths.go and about the JSON rules defined in
// the Concepts specification and the reference neondoll decoder.

package dollcard

import (
	"encoding/json"
	"strings"
)

// Validate produces the list of problems with the given Doll Card file map.
// Every entry in files is a canonical relative path.
func Validate(files map[string][]byte) []problem {
	var ps []problem

	// 1. Required files.
	for _, name := range requiredFiles {
		if _, ok := files[name]; !ok {
			ps = append(ps, Problem(name, "missing required file"))
		}
	}

	// 2. card.json — the version index.
	if data, ok := files["card.json"]; ok {
		_, hdrPs := parseHeader(data)
		ps = appendAll(ps, hdrPs)
	}

	// 3. identity.json — structured identity.
	if data, ok := files["identity.json"]; ok {
		_, idPs := parseIdentity(data)
		ps = appendAll(ps, idPs)
	}

	// 4. soul.md — prose Soul must be non-empty (whitespace does not count).
	if data, ok := files["soul.md"]; ok {
		if strings.TrimSpace(string(data)) == "" {
			ps = append(ps, Problem("soul.md", "must not be empty"))
		}
	}

	// 5. owner/owner.md — prose Owner must be non-empty.
	if data, ok := files["owner/owner.md"]; ok {
		if strings.TrimSpace(string(data)) == "" {
			ps = append(ps, Problem("owner/owner.md", "must not be empty"))
		}
	}

	// 6. Optional structured JSON areas, validated when present.
	for _, name := range []string{"self/self.json", "drives/drives.json", "goals/goals.json", "intentions/intentions.json"} {
		if data, ok := files[name]; ok {
			ps = appendAll(ps, checkJSON(name, data))
		}
	}

	// 7. Memory JSONL — each non-empty line must be one complete, valid
	//    Memory Record (a single JSON value). Blank lines are allowed.
	for name, data := range files {
		if !strings.HasPrefix(name, "memories/") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		ps = appendAll(ps, validateMemoryJSONL(name, data))
	}

	// 8. Every skill directory under skills/ must be a real Agent Skill:
	//    a directory rooted at a required SKILL.md (see Concepts spec).
	ps = appendAll(ps, validateSkills(files))

	return ps
}

// appendAll returns the concatenation of a and b.
func appendAll(a, b []problem) []problem {
	for _, p := range b {
		a = append(a, p)
	}
	return a
}

// checkJSON returns a problem if data is not valid JSON, else the empty list.
func checkJSON(name string, data []byte) []problem {
	if json.Valid(data) {
		return nil
	}
	return []problem{Problem(name, "malformed JSON")}
}

// validateMemoryJSONL checks every non-empty line of one memory segment is a
// complete, parseable JSON value.
func validateMemoryJSONL(name string, data []byte) []problem {
	var ps []problem
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !json.Valid([]byte(line)) {
			ps = append(ps, Problem(name, "malformed memory record (line not a complete JSON value)"))
		}
	}
	return ps
}

// validateSkills checks that any skill directory present under skills/ is a
// valid Agent Skill, i.e. it contains a SKILL.md. This preserves the v1 rule
// that a Skill is a directory rooted at a required SKILL.md, and rejects
// partial skill payloads rather than silently dropping them.
func validateSkills(files map[string][]byte) []problem {
	var ps []problem
	skills := make(map[string]bool)
	for name := range files {
		if !strings.HasPrefix(name, "skills/") {
			continue
		}
		rest := name[len("skills/"):]
		if i := strings.Index(rest, "/"); i > 0 {
			skills[rest[:i]] = true
		}
	}
	for dir := range skills {
		if _, ok := files["skills/"+dir+"/SKILL.md"]; !ok {
			ps = append(ps, Problem("skills/"+dir, "missing SKILL.md (each Agent Skill must be rooted at SKILL.md)"))
		}
	}
	return ps
}
