# कोश Kosha — Encrypted Terminal Notes Vault

**Kosha** (Sanskrit कोश: treasury / vault) is a keyboard-first, terminal-only encrypted notes application built in Go. All your notes are stored in encrypted `.vault` files using XChaCha20-Poly1305 authenticated encryption, derived from a single master passphrase via Argon2id.

## Features

- **End-to-end encryption**: Every file on disk is encrypted. Opening `.vault` files in any editor shows only binary gibberish.
- **Hierarchical organization**: Books → Chapters → Notes. Each chapter is a separate encrypted file.
- **Custom markup**: Bold, italic, underline, strikethrough, highlights, alignment, headings, tags, and inter-note links.
- **AI-powered refinement**: Grammar and clarity fixes via Gemini API, with diff review and accept/reject per block.
- **Fuzzy search**: Search across all notes by title, body, or tags.
- **Auto-lock**: Configurable idle timeout zeros key material and clears memory.
- **Snapshots**: Version history with encrypted snapshots before AI passes.
- **Glassmorphism UI**: "Indigo Ink & Saffron" theme with faux-glass panels.
- **Cross-platform**: Linux, macOS, Windows Terminal.

## Install

### From source

```bash
go install github.com/YaswanthKumarMallela01/kosha/cmd/kosha@latest
```

### Build from repo

```bash
git clone https://github.com/YaswanthKumarMallela01/KOSHA.git
cd KOSHA
go build -o kosha ./cmd/kosha
```

### Using Make

```bash
make build    # Build binary
make test     # Run tests
make lint     # Run go vet + golangci-lint
make run      # Build and run
```

## Configuration

### First Run

```bash
kosha init
```

This creates:
- `~/.kosha/config.json` — encrypted key check and Argon2 parameters
- `~/.kosha/books/inbox/` — default "Inbox" book with "Quick Capture" chapter

Override the data directory with `KOSHA_HOME` environment variable.

### Environment Variables

Create a `.env` file in your working directory or `~/.kosha/.env`:

```env
GEMINI_API_KEY=your-api-key-here
GEMINI_MODEL=gemini-2.0-flash
KOSHA_LOCK_MINUTES=5
```

- `GEMINI_API_KEY`: Required for AI refinement (`ctrl+g`). Get one from [Google AI Studio](https://aistudio.google.com/).
- `GEMINI_MODEL`: Gemini model name (default: `gemini-2.0-flash`).
- `KOSHA_LOCK_MINUTES`: Auto-lock timeout in minutes. Set to `0` to disable. Default: `5`.

## Commands

| Command | Description |
|---------|-------------|
| `kosha` | Launch the TUI at the library screen |
| `kosha init` | Initialize a new vault |
| `kosha add "text"` | Quick capture a note to Inbox / Quick Capture |
| `echo "idea" \| kosha add` | Pipe text as a quick capture |
| `kosha search "query"` | Launch TUI with search prefilled |
| `kosha export <book> [--chapter <title>] [--out <dir>]` | Export to Markdown |
| `kosha version` | Show version |
| `kosha help` | Show help |

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate lists |
| `Enter` | Open selected item |
| `Esc` / `Backspace` | Go back |
| `n` | New (book/chapter/note) |
| `r` | Rename |
| `d` | Delete (with confirmation) |
| `p` | Pin/unpin note |
| `s` | Toggle sort order |
| `/` or `Ctrl+K` | Search |
| `t` | Filter by tag |
| `e` | Edit note |
| `x` | Export to Markdown |
| `Ctrl+L` | Lock vault now |
| `Ctrl+G` | AI refine (Gemini) |
| `?` | Help overlay |
| `q` | Quit (from library screen) |

### Editor

| Key | Action |
|-----|--------|
| Arrow keys | Move cursor |
| `Shift+Arrows` | Select text |
| `Ctrl+Left/Right` | Word jump |
| `Home/End` | Start/end of line |
| `PgUp/PgDn` | Page scroll |
| `Ctrl+Z` | Undo |
| `Ctrl+Y` | Redo |
| `Ctrl+C` | Copy |
| `Ctrl+X` | Cut |
| `Ctrl+V` | Paste |
| `Ctrl+R` | Toggle edit/preview mode |
| `Ctrl+F` | Toggle format pane |
| `Alt+B` | Toggle bold |
| `Alt+I` | Toggle italic |
| `Alt+U` | Toggle underline |
| `Alt+S` | Toggle strikethrough |
| `Alt+L` | Align left |
| `Alt+E` | Align center |
| `Alt+R` | Align right |

### Diff Review

| Key | Action |
|-----|--------|
| `y` | Accept block |
| `n` | Reject block |
| `a` | Accept all |
| `x` | Reject all |
| `k` | Keep as-is (mark processed) |
| `Tab/Shift+Tab` | Next/previous block |
| `Enter` | Apply changes |
| `Esc` | Cancel |

## Markup Grammar

Kosha uses a custom inline markup format stored in note bodies:

| Syntax | Rendering |
|--------|-----------|
| `**text**` | **Bold** |
| `*text*` | *Italic* |
| `__text__` | <u>Underline</u> |
| `~~text~~` | ~~Strikethrough~~ |
| `==text==` | Important word (saffron bold) |
| `^^text^^` | Important sentence (saffron bar + underline) |
| `` `code` `` | Inline code |
| `# Heading` | Heading level 1 |
| `## Heading` | Heading level 2 |
| `### Heading` | Heading level 3 |
| `> quote` | Blockquote |
| `#tag` | Tag (clickable, searchable) |
| `[[Note Title]]` | Inter-note link |
| `:::left` | Left-align following text |
| `:::center` | Center-align following text |
| `:::right` | Right-align following text |
| `\*` | Escape (literal asterisk) |

### Nesting

Markup can be nested: `**bold *and italic***` renders as bold text with an italic portion.

### Tags

Tags use `#` followed by letters, digits, hyphens, or underscores (e.g., `#project-x`, `#meeting_notes`). Tags are extracted for the tag filter and search index.

### Links

`[[Note Title]]` creates a navigable link to another note. In reading mode, `Tab` cycles through links and `Enter` follows them. Resolution prefers: same chapter → same book → anywhere. If ambiguous, a picker is shown.

## Security Model

### What encryption protects

- **At rest**: All note content, titles, chapter titles, book names, and metadata are encrypted in `.vault` files. Opening them in any editor shows only binary ciphertext.
- **Key derivation**: Argon2id with configurable parameters (default: time=3, memory=64 MiB, threads=4) protects against brute-force attacks.
- **Per-file keys**: Each file has a unique random salt. HKDF-SHA256 derives per-file subkeys from the master key.
- **Tamper detection**: The entire file header is passed as AEAD additional data. Flipping any byte causes authentication failure.
- **In memory**: Key material is zeroed on lock/exit. Auto-lock clears all cached plaintext.

### What encryption does NOT protect

- **Side channels**: A determined attacker with access to your running process memory can extract keys.
- **File metadata**: File sizes, modification times, and directory structure are visible (though names are non-descriptive ULIDs).
- **Gemini AI**: When using `Ctrl+G`, the plaintext of changed blocks is sent to Google's Gemini API. This is by design for grammar/highlight processing. **Do not use AI refinement for highly sensitive content.**
- **Clipboard**: Copied text goes through the system clipboard, which other applications can read.
- **Swap/hibernation**: OS may write memory pages to disk.

### ⚠ WARNING

**If you lose your passphrase, your data is permanently lost.** There is no recovery mechanism. Consider keeping a secure backup of your passphrase.

## Glassmorphism (Faux)

Kosha's UI uses a "faux glassmorphism" effect since terminals cannot blur backgrounds:

- **Glass panels**: Dark fill color (`#1A1E3C`) with rounded borders (`#2B3166`), a lighter frost edge at the top (`#3A4180`), and a shadow row beneath (`#090A18`).
- **Gradient background**: Vertical gradient from midnight indigo (`#0E1022`) to slightly lighter (`#161A38`).
- **Truecolor**: The theme uses 24-bit colors. Set `COLORTERM=truecolor` if your terminal supports it.

For the closest-to-real glass effect, enable your terminal's transparency/acrylic:

- **Windows Terminal**: Settings → Profiles → Appearance → Enable acrylic, set opacity to ~85%
- **WezTerm**: `window_background_opacity = 0.85`
- **Kitty**: `background_opacity 0.85`
- **iTerm2**: Profiles → Window → Transparency

## Data Layout

```
~/.kosha/
├── config.json              # Encrypted key check, Argon2 params, master salt
├── .env                     # Optional: API keys, settings
└── books/
    ├── inbox/
    │   ├── book.meta        # Encrypted book metadata
    │   ├── 01JXYZ...vault   # Encrypted chapter (ULID filename)
    │   ├── 01JXYZ...vault
    │   └── .snapshots/
    │       └── 01JXYZ.../
    │           ├── 20261002T083000.vault
    │           └── 20261002T093000.vault
    └── my-project/
        ├── book.meta
        └── 01JXYZ...vault
```

## License

MIT
