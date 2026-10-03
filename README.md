# कोश Kosha — Encrypted Terminal Notes Vault

**Kosha** (Sanskrit कोश: treasury / vault) is a keyboard-first, terminal-only encrypted notes vault built in Go. All notes are organized cleanly into **Books** and stored in encrypted `.vault` files using **XChaCha20-Poly1305** authenticated encryption with keys derived via **Argon2id**.

---

## 💡 Streamlined Architecture: Book ➜ Vaults (Side-wise)

Kosha simplifies your notes workflow into a fast, direct structure:

1. **📚 BOOK (Shelf of Projects in `D:\books\`)**
   - Each book is a dedicated folder inside `D:\books\<book-slug>\` (e.g. `D:\books\operating-systems\`).
   - Lined up side-by-side with neat, compact 3D ASCII book cards.

2. **🔐 VAULT SECTIONS (Encrypted Chapters inside the Book)**
   - Inside any book, your **Vaults are lined up side-wise** (`vault1`, `vault2`, `vault3`...) with matching 3D ASCII vault cards.
   - Each `.vault` file is an encrypted chapter (`<ulid>.vault`) where you write your notes directly.
   - **No separate intermediate notes layer**: selecting a vault and pressing `Enter` or `E` opens the coding editor immediately so you can write and organize notes inside that vault right away!

---

## 📦 Downloads & Installation (Windows, macOS, Linux)

Kosha is available as a single, static binary with **zero external dependencies** for Windows, macOS, and Linux.

### Download from GitHub Releases
Download the precompiled binary for your operating system directly from the **[GitHub Releases](https://github.com/YaswanthKumarMallela01/KOSHA/releases)** page:

| Platform | Binary | Architecture |
|---|---|---|
| **Windows** | `kosha-windows-amd64.exe` | 64-bit Intel/AMD |
| **Linux** | `kosha-linux-amd64` | 64-bit x86_64 |
| **macOS (Apple Silicon)** | `kosha-darwin-arm64` | M1 / M2 / M3 / M4 |
| **macOS (Intel)** | `kosha-darwin-amd64` | 64-bit Intel |

#### Quick Install (from source with Go)
If you have Go installed:
```bash
go install github.com/YaswanthKumarMallela01/kosha/cmd/kosha@latest
```
Ensure `~/go/bin` (or `%USERPROFILE%\go\bin` on Windows) is in your `PATH`.

---

## 🎨 Compact ASCII Art & Pitch-Black Developer Theme

- **Pure Pitch Black (`#000000`)**: Zero murky blues/purples. Minimalist, high-contrast aesthetic across all screens.
- **Compact 3D ASCII Book & Vault Cards**: Neat, matching 24x7 terminal cards lined up horizontally side-wise with sliding navigation (`◀◀ [1 of 4] ▶▶`).
- **Live Vault Preview**: Selecting any vault immediately displays an encrypted file summary and a live text preview of the notes written inside it.
- **Peaceful Read Mode**: Distraction-free reading with beautiful Markdown ANSI rendering (bold, italics, underline, highlights, headers), with minimal peaceful hints.
- **Spacious Coding Editor**: Line numbers with vertical dividers, active line highlights, and a status bar cleanly pinned to the **very bottom** of the pane leaving full space for text.
- **Golden Saffron Highlights**: Active selections, book titles, headers, and key markers highlighted in `#F5A623`.
- **Zero Green Anywhere**: Strictly respects the saffron, lotus pink, and body text palette.
- **Zero Terminal Scrolling**: Auto-clamped layouts prevent duplicate footers or screen tearing on Windows Terminal.

---

## 🤖 Gemini AI Copy-Editor (`gemini-2.0-flash-lite`)

Set your API key and preferred model in `.env` (or system environment):
```env
GEMINI_API_KEY=your_gemini_api_key_here
GEMINI_MODEL=gemini-2.0-flash-lite
```
Inside the editor, press **`Ctrl+G`** to automatically copy-edit spelling, grammar, and add sparse, beautiful highlights (`==key points==`).

---

## ⌨️ Controls & Keybindings

### Navigation (Universal)
- **`← / →` or `h / l` / `↑ / ↓`**: Move selection across side-wise cards smoothly
- **`Enter`**: Open selected Book, or open Vault in **Peaceful Read Mode**
- **`E`**: Open Vault directly in **Edit Mode**
- **`Esc` or `Backspace`**: Go back to previous screen
- **`/` or `Ctrl+K`**: Fuzzy search across all books, vaults, and notes
- **`t`**: Filter notes by tag
- **`s`**: Toggle sort order (newest / oldest)
- **`Ctrl+L`**: Lock vault immediately (clears master key from memory)
- **`?`**: Toggle full keyboard reference guide
- **`q`**: Quit application (from library screen)

### Book & Vault Management
- **`Ctrl+N` or `n` / `N`**: Create a new Book (on Library screen) or new Vault (inside a Book)
- **`r`**: Rename selected Book or Vault
- **`d`**: Delete selected item (strictly requires master passphrase confirmation)
- **`x`**: Export decrypted content to standard Markdown files

### Reading & Editing
- **`Enter` (on Vault)**: Open **Peaceful Read Mode** with rendered typography
- **`E` (in Read Mode or on Vault)**: Switch to **Coding Editor** to write notes
- **Two-Finger / Wheel Scroll**: Smooth mouse trackpad scrolling in both Read and Edit modes
- **`Shift + ↑ / ↓ / ← / →`**: Select multiple lines or characters with cursor
- **Mouse Click & Drag**: Highlight and select text across multiple lines with mouse/trackpad
- **`Ctrl+A`**: Select entire note across all lines
- **`Ctrl+F`**: Open formatting toolbar (applies bold, italic, underline, strike, highlights, quotes, headings across entire selection)
- **`Alt+B / Alt+I / Alt+U / Alt+S`**: Instant inline formatting for single words or multi-line selections
- **`Ctrl+I` (in Editor)**: Insert embedded image `!img[alt](path|align|width%)` with alignment & sizing
- **`Ctrl+P` (in Read/Edit Mode)**: Export note to styled **PDF** (passphrase-protected)
- **`Ctrl+W` (in Read/Edit Mode)**: Export note to Microsoft **Word (.docx)** (passphrase-protected)
- **`Ctrl+G`**: Run Gemini AI copy-editor & smart highlights
- **`Ctrl+S`**: Save manual snapshot of vault file
- **`Esc`**: Save note and exit back to the side-wise vaults shelf

---

## 🖼️ Image Support & Rich Formatting

- **Embedded Images**: Insert images using the syntax `!img[caption](filepath|align|width%)` (e.g. `!img[Diagram](arch.png|center|80%)`). Supports `left`, `center`, and `right` alignments with custom percentage sizing. Rendered neatly with framed symbol placeholders in terminal read mode and included in exports.
- **Electric Blue Links**: Pasted URLs (`https://...`, `http://...`, `www....`), wiki links (`[[...]]`), and Markdown links (`[text](url)`) automatically turn vibrant blue (`#38BDF8`) with underline in both edit mode, read mode, PDF export, and Word export.
- **Robust Multi-line Text Selection**: Select multiple lines or words using `Shift+Arrows`, `Ctrl+A`, or mouse drag. Pressing `Ctrl+F` keeps selection intact and lets you apply bold, italics, highlights, quotes, and alignments across the entire selection.
- **High-Visibility Typography**: Clean, high-contrast formatting (pure white bold, soft lavender italic, saffron highlights, gold underlines, and distinct code blocks) that stands out clearly against the pitch-black backdrop.
- **Passphrase-Protected Export**: Export full books, chapters, or individual notes to standard **Markdown**, professional **PDF**, or Microsoft **Word (.docx)** with identical alignment and formatting preserved. All exports require master passphrase authentication.

---

## 🛠️ CLI Commands

| Command | Description |
|---|---|
| `kosha` | Launch Kosha at the Books selector |
| `kosha vault` | Explicit command to view all books and select which one to write in |
| `kosha add "text"` | Quick capture a note directly to your vault without opening the TUI |
| `echo "idea" \| kosha add` | Pipe text from stdin directly into quick capture |
| `kosha search "query"` | Launch directly into search mode with prefilled query |
| `kosha export <book> [--format pdf\|word\|md] [--out <dir>]` | Export book notes to PDF, Word, or Markdown |
| `kosha version` | Print version (v0.3.2) |
| `kosha help` | Show command reference |

---

## 🔒 Security Architecture

- **Cipher**: XChaCha20-Poly1305 with 24-byte random nonces and 16-byte random salts per file.
- **KDF**: Argon2id (time=3, memory=64 MiB, threads=4) deriving master key from passphrase once per session.
- **Subkeys**: HKDF-SHA256 deriving distinct per-file encryption keys.
- **Header Authentication**: Format magic (`KOSHAVLT`) and parameters authenticated as AEAD Additional Data; any tampering causes immediate decryption failure.
- **Memory Security**: Passphrases and master keys are actively zeroed in memory on lock or application exit.

---

## 📄 License
MIT License
