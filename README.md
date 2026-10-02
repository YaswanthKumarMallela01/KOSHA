# कोश Kosha — Encrypted Terminal Notes Vault

**Kosha** (Sanskrit कोश: treasury / vault) is a keyboard-first, terminal-only encrypted notes vault built in Go. All notes are organized cleanly into **Books** and stored in encrypted `.vault` files using **XChaCha20-Poly1305** authenticated encryption with keys derived via **Argon2id**.

---

## 💡 Mental Model: What is a Book, Vault File, and Note?

To keep your notes clean and organized, Kosha uses a clear three-tier hierarchy:

1. **📚 BOOK (Directory on Disk)**
   - A folder directly inside `D:\books\` (e.g. `D:\books\algorithms\`).
   - Represents a notebook, subject, or project.
   - Contains its own individual encrypted `.vault` files and metadata.

2. **🔐 VAULT FILE / CHAPTER (`.vault` File)**
   - An individual encrypted binary file inside a book (e.g. `D:\books\algorithms\01J9X4...vault`).
   - The chapter title and all content inside are 100% encrypted. Opening it in Notepad shows only binary gibberish.

3. **📝 NOTE (Page / Entry)**
   - An entry stored inside the vault file with a title, formatted body, timestamps, pin status, and tags.

---

## 🚀 Universal Access

Kosha is installed universally on your machine:
```bash
go install ./cmd/kosha
```
Because `C:\Users\<user>\go\bin` is in your PATH, you can open a terminal in **any folder or drive** and simply run:
```bash
kosha vault      # Opens the book selector
# or
kosha            # Launches Kosha library
```

All data is safely persisted in one single place:
**`D:\books\`** (individual book folders containing their own vault files).

---

## 🎨 Terminal Book Aesthetic & Pitch-Black Developer Theme

- **Pure Pitch Black (`#000000`)**: Zero murky blues/purples. Minimalist, high-contrast aesthetic across all screens.
- **ASCII Art Book Shapes (`kosha vault`)**: Realistic physical terminal book shapes with your custom title emblazoned directly on the book spine and cover, with book metadata clearly displayed underneath.
- **Coding Text Editor**: Clean line numbers with vertical dividers, active line highlights, live cursor position, line count, word count, character count, and instant autosave indicators.
- **Golden Saffron Highlights**: Active selections, book titles, headers, and key markers highlighted in `#F5A623`.
- **Zero Green Anywhere**: Strictly respects the saffron, lotus pink, and body text palette.
- **Zero Terminal Scrolling**: Auto-clamped layouts prevent duplicate footers or screen tearing on Windows Terminal.

---

## ⌨️ Controls & Keybindings

### Navigation (Universal)
- **`↑ / ↓` or `k / j`**: Move selection up and down smoothly
- **`Enter`**: Open selected Book / Vault / Note
- **`Esc` or `Backspace`**: Go back to previous screen
- **`/` or `Ctrl+K`**: Fuzzy search across all books, vaults, and notes
- **`t`**: Filter notes by tag
- **`s`**: Toggle sort order (newest / oldest)
- **`Ctrl+L`**: Lock vault immediately (clears master key from memory)
- **`?`**: Toggle full keyboard reference guide
- **`q`**: Quit application (from library screen)

### Book & Vault Management
- **`Ctrl+N` or `n` / `N`**: Create a new Book (on Library screen) or new Vault File (inside a Book) or new Note (inside a Vault)
- **`r`**: Rename selected Book, Vault, or Note
- **`d`**: Delete selected item (with confirmation prompt)
- **`p`**: Pin / unpin note to top of chapter
- **`x`**: Export decrypted content to standard Markdown files

### Coding Editor Mode
- **Arrow keys**: Move cursor
- **Shift + Arrows**: Select text
- **Ctrl + Left/Right**: Jump by word
- **Ctrl + Z / Ctrl + Y**: Undo / Redo
- **Ctrl + C / X / V**: Copy / Cut / Paste
- **Ctrl + R**: Toggle between Edit mode and formatted Preview mode
- **Ctrl + F**: Open formatting toolbar
- **Ctrl + G**: Run Gemini AI copy-editor & sparse highlight refinement (syncs active buffer)
- **Ctrl + S**: Save manual snapshot of vault file
- **Esc**: Save note and exit back to reading view

---

## 🛠️ CLI Commands

| Command | Description |
|---|---|
| `kosha` | Launch Kosha at the Books selector |
| `kosha vault` | Explicit command to view all books and select which one to write in |
| `kosha add "text"` | Quick capture a note directly to your vault without opening the TUI |
| `echo "idea" \| kosha add` | Pipe text from stdin directly into quick capture |
| `kosha search "query"` | Launch directly into search mode with prefilled query |
| `kosha export <book>` | Export book notes to Markdown directory |
| `kosha version` | Print version |
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
