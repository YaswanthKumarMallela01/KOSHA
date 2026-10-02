# Decisions

## Go Version
- Spec requested Go 1.27.1; Go 1.27.1 is installed and used. The `go.mod` uses `go 1.27.1`.

## Gemini Client
- The spec allowed using either `google.golang.org/genai` or raw REST API via `net/http`. We use the REST API approach for simplicity, fewer dependencies, and direct control over structured output JSON schemas. The Gemini REST endpoint `generativelanguage.googleapis.com/v1beta/models/{model}:generateContent` is used with `responseMimeType: application/json` and a `responseSchema` for structured output.

## Module Path
- Module path is `github.com/YaswanthKumarMallela01/kosha` matching the user's GitHub remote.

## ULID Library
- Using `github.com/oklog/ulid/v2` as specified. ULIDs sort by creation time naturally.

## Clipboard
- System clipboard uses `github.com/atotto/clipboard`. On systems where this isn't supported (headless Linux without xclip/xsel), the editor falls back to an internal clipboard buffer.

## Editor Component
- Built a fully custom `internal/editor` component since Bubble Tea's stock `textarea` lacks selection and inline formatting. The editor implements rune-aware buffer management, undo/redo, selection, and format toggling.

## Markup Parser
- The markup parser uses a recursive-descent approach for inline formatting, processing line-by-line for block-level elements (headings, alignment directives, blockquotes). Nesting is supported (e.g., bold inside italic). Unmatched markers are treated as literal text.

## Tag Extraction
- Tags are extracted with a regex pattern that ensures `#tag` is recognized but `## headings` (with a space after #) are not. A tag starts with `#` followed by letters, digits, hyphens, or underscores, and must not be preceded by another `#`.

## Auto-lock
- Auto-lock zeroes the master key in memory and clears all cached chapters. The TUI then shows the passphrase prompt. Default timeout is 5 minutes, configurable via `KOSHA_LOCK_MINUTES` in `.env` (0 disables).

## Snapshot Format
- Snapshots are stored as encrypted `.vault` files at `<book>/.snapshots/<chapter-id>/<timestamp>.vault`. The timestamp format is `20060102T150405` for sortability. Maximum 20 snapshots per chapter; oldest are pruned on save.

## Atomic Writes
- All vault writes go through `AtomicWrite`: write to `.tmp`, fsync, rename. A `.bak` of the previous version is kept. File permissions are 0600, directories 0700.

## Color Theme
- No green is used anywhere in the UI. Success indicators use parchment or lotus pink. The "Indigo Ink & Saffron" theme is defined centrally in `internal/ui/theme.go` for easy tuning.

## Glassmorphism
- Terminal glassmorphism is faux: glass panels use a dark fill color (#1A1E3C) over a gradient background (#0E1022 to #161A38), with rounded borders, a frost edge highlight, and a shadow row beneath. True glass-like effects require the terminal emulator's own transparency settings (Windows Terminal acrylic, Kitty, WezTerm).

## AI Highlight Density
- Highlights are limited to ~15% of block text. Excess highlights are removed automatically from the end of the text. The system instruction for Gemini emphasizes sparing highlight usage.

## Edit Distance Threshold
- If word-level edit distance between original and AI-refined text exceeds 35%, the block is flagged as "heavily changed" in the diff review UI.

## Key Derivation Strategy
- One master passphrase → Argon2id → master key (once per session). Per-file subkeys via HKDF-SHA256 using each file's random salt. This avoids running slow Argon2 for every chapter file.
