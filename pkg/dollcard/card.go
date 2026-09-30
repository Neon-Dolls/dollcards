// Package dollcard — core Doll Card model, validation, and (de)serialization.
//
// This package is the implementation authority for the Doll Card interchange
// format within the dollcards repository. It mirrors the encoding/decoding
// behavior of the reference implementation in Neon-Dolls/neondoll but is
// deliberately Core-free: it models signed Doll Card files and directories,
// not Doll Core runtime or Persistence internals.
//
// The specification authority for Doll Card is:
//
//	dividebyzero/NeonDoll-Concepts/dollcard.md

package dollcard

import (
	"encoding/json"
	"strconv"
)

// currentCardVersion is the one Doll Card container/semantic version supported
// by this implementation. NeonDoll version numbers are monotonically
// increasing integers, so version 1 is integer 1 (not a semver).
const currentCardVersion = 1

// cardHeader is the JSON shape of card.json, the Doll Card index/metadata.
// Version 1 carries only the format version so that a reader can decide
// whether it understands the container.
type cardHeader struct {
	Version int `json:"version"`
}

// identity is the JSON shape of identity.json.
//
// Doll Card version 1 stores the structured identifying/provenance
// information of the Doll (which Doll is this?) separately from personality,
// self-model, ownership, and presentation (which live in soul.md/self/etc.).
type identity struct {
	DollID        string `json:"doll_id"`
	CanonicalName string `json:"canonical_name"`
	TemplateRef   string `json:"template_ref,omitempty"`
}

// requiredFiles are the files that must exist (and survive validation) in
// every valid Doll Card, mirroring the requirement in the reference decoder.
var requiredFiles = []string{
	"card.json",
	"identity.json",
	"soul.md",
	"owner/owner.md",
}

// problem describes a single validation failure in human-readable terms.
type problem struct {
	Path string // card-relative path the problem concerns ("" for whole-card)
	Msg  string // human-readable description
}

// Problem constructs a single problem.
func Problem(path, msg string) problem {
	return problem{Path: path, Msg: msg}
}

// parseIdentity parses identity.json bytes and checks the version-1 required
// fields. It returns the identity and any problems found.
func parseIdentity(data []byte) (identity, []problem) {
	var id identity
	if err := json.Unmarshal(data, &id); err != nil {
		return id, []problem{Problem("identity.json", "not valid JSON: "+err.Error())}
	}
	var ps []problem
	if id.DollID == "" {
		ps = append(ps, Problem("identity.json", "missing doll_id"))
	}
	if id.CanonicalName == "" {
		ps = append(ps, Problem("identity.json", "missing canonical_name"))
	}
	return id, ps
}

// parseHeader parses card.json and checks that the version is supported.
// It returns the header and any problems found.
func parseHeader(data []byte) (cardHeader, []problem) {
	header := cardHeader{Version: 0}
	if !json.Valid(data) {
		return header, []problem{Problem("card.json", "not valid JSON")}
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return header, []problem{Problem("card.json", "cannot unmarshal index: "+err.Error())}
	}
	if header.Version != currentCardVersion {
		return header, []problem{Problem("card.json", "unsupported card version "+strconv.Itoa(header.Version)+" (supported: "+strconv.Itoa(currentCardVersion)+")")}
	}
	return header, nil
}
