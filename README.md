# DollCard

A DollCard is a portable, self-contained digital identity package for NeonDoll companions. It encapsulates a doll's essential data—including identity, memories, skills, and configuration—into a single, versioned archive that can be backed up, transferred, and loaded into compatible hosts.

## What is a DollCard?

A DollCard is a version‑1 `.dollcard` file, which is a ZIP archive with a well‑defined internal layout:

```
card.json
identity.json
soul.md
owner/owner.md
<optional areas …>
```

Each entry is stored at its canonical relative path (no `./`, no `//`, no `..`, no drive letters, no symlinks). The archive is produced and consumed by the `dollcard` command‑line tool.

## Toolkit Commands

The `dollcard` CLI provides five primary verbs:

### `dollcard create <dir> [flags]`

Initialize a new DollCard in `<dir>`. The directory must not exist or must be empty. On success, a valid DollCard skeleton is written atomically.

Flags:
- `-name <string>`: Optional doll name (defaults to a generated UUID).
- `-id <string>`: Optional doll ID (UUID‑v4); if omitted a cryptographically random ID is generated.

### `dollcard validate <dir>|<file.dollcard>`

Validate a DollCard directory or archive. Returns a list of problems (missing required files, invalid JSON, etc.). Exit code 0 if valid, 1 if invalid.

### `dollcard compress <dir> <output.dollcard>`

Create a `.dollcard` archive from a valid DollCard directory. The directory is validated first; compression fails if validation fails.

### `dollcard extract <archive.dollcard> <dir>`

Extract a `.dollcard` archive into `<dir>`. The destination must not exist or must be empty. After extraction, the result is validated.

### `dollcard migrate <dir>|<file.dollcard>`

Upgrade a DollCard from an older layout to the current version (if needed). Currently a no‑op for version‑1.

## Typical Workflow

```
# 1. Create a new doll
dollcard create -name "Aiko" ./mydoll

# 2. Edit the doll's files (soul.md, memories/, skills/, …)
#    … make changes …

# 3. Validate your edits
dollcard validate ./mydoll

# 4. Pack for transport or backup
dollcard compress ./mydoll ./mydoll.dollcard

# 5. Later, restore and continue
dollcard extract ./mydoll.dollcard ./restored
dollcard validate ./restored   # should be clean
```

## Relationship to NeonDoll Core

A DollCard is the **portable snapshot** of a NeonDoll companion’s persistent state. NeonDoll Core (the runtime) loads a DollCard into memory at startup and writes changes back to the DollCard on clean shutdown. The DollCard format is defined in the [NeonDoll Concepts repository](https://github.com/dividebyzero/NeonDoll-Concepts); this toolkit is the reference implementation.

## Examples

See the `examples/` directory for canonical DollCards:
- `examples/minimal/` – a minimal valid DollCard produced by `dollcard create`
- `examples/full/`   – a richer example with optional areas populated

## Contributing

See `CONTRIBUTING.md` for details on the development workflow, testing, and CI.

## License

MIT – see the `LICENSE` file.