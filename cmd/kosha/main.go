package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/term"

	"github.com/YaswanthKumarMallela01/kosha/internal/ai"
	"github.com/YaswanthKumarMallela01/kosha/internal/config"
	"github.com/YaswanthKumarMallela01/kosha/internal/crypto"
	"github.com/YaswanthKumarMallela01/kosha/internal/export"
	"github.com/YaswanthKumarMallela01/kosha/internal/markup"
	"github.com/YaswanthKumarMallela01/kosha/internal/search"
	"github.com/YaswanthKumarMallela01/kosha/internal/store"
	"github.com/YaswanthKumarMallela01/kosha/internal/ui"
)

const version = "0.3.3"

func main() {
	lipgloss.SetColorProfile(termenv.TrueColor)
	if len(os.Args) < 2 {
		runTUI("")
		return
	}

	switch os.Args[1] {
	case "vault":
		runTUI("")
	case "init":
		runInit()
	case "add":
		runAdd()
	case "search":
		query := ""
		if len(os.Args) > 2 {
			query = strings.Join(os.Args[2:], " ")
		}
		runTUI(query)
	case "export":
		runExport()
	case "version":
		fmt.Printf("Kosha v%s\n", version)
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'kosha help' for usage.\n", os.Args[1])
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`कोश Kosha — Encrypted Terminal Notes Vault

Usage:
  kosha              Launch the TUI at the library screen
  kosha init         Initialize a new vault (create config, set passphrase)
  kosha add "text"   Quick capture a note to Inbox (also supports piping)
  kosha search "q"   Launch TUI with search prefilled
  kosha export <book> [--chapter <title>] [--out <dir>]
                     Export decrypted notes to Markdown files
  kosha version      Show version
  kosha help         Show this help message

Key Bindings (in TUI):
  ↑/↓ or j/k    Navigate lists
  Enter          Open selected item
  Esc/Backspace  Go back
  n              New (book/chapter/note)
  e              Edit note
  r              Rename
  d              Delete (with confirmation)
  p              Pin/unpin note
  s              Toggle sort order
  / or Ctrl+K    Search
  t              Filter by tag
  x              Export to Markdown
  Ctrl+G         AI refine (Gemini)
  Ctrl+L         Lock vault now
  ?              Help overlay
  q              Quit (from library screen)`)
}

// readPassphrase reads a passphrase from the terminal with no echo.
func readPassphrase(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	pass, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // newline after hidden input
	if err != nil {
		return nil, fmt.Errorf("reading passphrase: %w", err)
	}
	return pass, nil
}

func runInit() {
	cfg, err := config.LoadConfig()
	if err == nil && cfg.MasterSalt != nil && len(cfg.MasterSalt) > 0 {
		fmt.Println("Kosha vault already initialized.")
		fmt.Println("Data directory:", cfg.DataDir)
		return
	}

	fmt.Println("कोश Kosha — Initializing New Vault")
	fmt.Println("====================================")
	fmt.Println()
	fmt.Println("⚠  WARNING: If you lose your passphrase, your data is permanently lost.")
	fmt.Println("   There is no recovery mechanism.")
	fmt.Println()

	pass1, err := readPassphrase("Enter passphrase: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if len(pass1) < 8 {
		fmt.Fprintln(os.Stderr, "Error: passphrase must be at least 8 characters.")
		os.Exit(1)
	}

	pass2, err := readPassphrase("Confirm passphrase: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if string(pass1) != string(pass2) {
		fmt.Fprintln(os.Stderr, "Error: passphrases do not match.")
		os.Exit(1)
	}

	dataDir, err := config.DefaultDataDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error determining data directory: %v\n", err)
		os.Exit(1)
	}

	if err := config.EnsureDataDir(dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating data directory: %v\n", err)
		os.Exit(1)
	}

	masterSalt, err := crypto.GenerateSalt()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating salt: %v\n", err)
		os.Exit(1)
	}

	params := crypto.DefaultParams()
	masterKey := crypto.DeriveMasterKey(pass1, masterSalt, params)
	defer crypto.ZeroBytes(masterKey)
	crypto.ZeroBytes(pass1)
	crypto.ZeroBytes(pass2)

	keyCheck, err := crypto.CreateKeyCheck(masterKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating key check: %v\n", err)
		os.Exit(1)
	}

	newCfg := &config.Config{
		DataDir:     dataDir,
		MasterSalt:  config.Base64Bytes(masterSalt),
		ArgonParams: params,
		KeyCheck:    config.Base64Bytes(keyCheck),
	}

	if err := config.SaveConfig(newCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	// Create Inbox book
	lib := store.NewLibrary(dataDir, masterKey)
	if err := lib.EnsureInbox(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Inbox: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✓ Vault initialized successfully!")
	fmt.Printf("  Data directory: %s\n", dataDir)
	fmt.Println("  Default book 'Inbox' with chapter 'Quick Capture' created.")
	fmt.Println()
	fmt.Println("Run 'kosha' to launch the TUI.")
}

func runAdd() {
	var text string

	// Check if there's piped input
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Piped input
		reader := bufio.NewReader(os.Stdin)
		data, err := io.ReadAll(reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		text = strings.TrimSpace(string(data))
	} else if len(os.Args) > 2 {
		text = strings.Join(os.Args[2:], " ")
	} else {
		fmt.Fprintln(os.Stderr, "Usage: kosha add \"your note text\"")
		fmt.Fprintln(os.Stderr, "  or:  echo \"your note\" | kosha add")
		os.Exit(1)
	}

	if text == "" {
		fmt.Fprintln(os.Stderr, "Error: empty note text.")
		os.Exit(1)
	}

	cfg, masterKey, err := unlockVault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer crypto.ZeroBytes(masterKey)

	lib := store.NewLibrary(cfg.DataDir, masterKey)
	if err := lib.LoadBooks(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading library: %v\n", err)
		os.Exit(1)
	}

	if err := lib.QuickCapture(text); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving note: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Note captured to Inbox / Quick Capture")
}

func runExport() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: kosha export <book-slug> [--format pdf|word|md] [--chapter <title>] [--out <dir>]")
		os.Exit(1)
	}

	bookSlug := os.Args[2]
	var chapterTitle, outDir string
	format := "md"

	for i := 3; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--format", "-f":
			if i+1 < len(os.Args) {
				format = strings.ToLower(os.Args[i+1])
				i++
			}
		case "--chapter":
			if i+1 < len(os.Args) {
				chapterTitle = os.Args[i+1]
				i++
			}
		case "--out":
			if i+1 < len(os.Args) {
				outDir = os.Args[i+1]
				i++
			}
		}
	}

	if outDir == "" {
		outDir = "."
	}

	cfg, masterKey, err := unlockVault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer crypto.ZeroBytes(masterKey)

	lib := store.NewLibrary(cfg.DataDir, masterKey)
	if err := lib.LoadBooks(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading library: %v\n", err)
		os.Exit(1)
	}

	chapters, err := lib.LoadChapters(bookSlug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading chapters: %v\n", err)
		os.Exit(1)
	}

	exported := 0
	for _, ch := range chapters {
		if chapterTitle != "" && ch.Title != chapterTitle {
			continue
		}

		for _, note := range ch.Notes {
			switch format {
			case "pdf":
				filename := fmt.Sprintf("%s/%s_%s.pdf", outDir, bookSlug, sanitizeFilename(note.Title))
				if err := export.ExportNoteToPDF(note, filename); err != nil {
					fmt.Fprintf(os.Stderr, "Error exporting %s to PDF: %v\n", filename, err)
					continue
				}
			case "word", "docx":
				filename := fmt.Sprintf("%s/%s_%s.docx", outDir, bookSlug, sanitizeFilename(note.Title))
				if err := export.ExportNoteToDOCX(note, filename); err != nil {
					fmt.Fprintf(os.Stderr, "Error exporting %s to DOCX: %v\n", filename, err)
					continue
				}
			default:
				filename := fmt.Sprintf("%s/%s_%s.md", outDir, bookSlug, sanitizeFilename(note.Title))
				md := fmt.Sprintf("# %s\n\n", note.Title)
				md += markup.RenderToMarkdown(note.Body)
				md += fmt.Sprintf("\n\n---\n*Created: %s | Updated: %s*\n",
					note.CreatedAt.Format("2006-01-02 15:04"),
					note.UpdatedAt.Format("2006-01-02 15:04"))

				if err := os.WriteFile(filename, []byte(md), 0600); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", filename, err)
					continue
				}
			}
			exported++
		}
	}

	fmt.Printf("✓ Exported %d notes [%s] to %s\n", exported, strings.ToUpper(format), outDir)
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_",
		"|", "_", " ", "_",
	)
	s := replacer.Replace(name)
	if len(s) > 50 {
		s = s[:50]
	}
	return s
}

func unlockVault() (*config.Config, []byte, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("loading config: %w (have you run 'kosha init'?)", err)
	}

	if cfg.MasterSalt == nil || len(cfg.MasterSalt) == 0 {
		return nil, nil, fmt.Errorf("vault not initialized; run 'kosha init' first")
	}

	maxAttempts := 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pass, err := readPassphrase("Enter passphrase: ")
		if err != nil {
			return nil, nil, fmt.Errorf("reading passphrase: %w", err)
		}

		masterKey := crypto.DeriveMasterKey(pass, []byte(cfg.MasterSalt), cfg.ArgonParams)
		crypto.ZeroBytes(pass)

		if crypto.ValidateKeyCheck([]byte(cfg.KeyCheck), masterKey) {
			return cfg, masterKey, nil
		}

		crypto.ZeroBytes(masterKey)
		remaining := maxAttempts - attempt
		if remaining > 0 {
			fmt.Fprintf(os.Stderr, "Wrong passphrase. %d attempts remaining.\n", remaining)
		}
	}

	return nil, nil, fmt.Errorf("too many failed attempts")
}

func runTUI(searchQuery string) {
	// Trap signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	cfg, masterKey, err := unlockVault()
	if err != nil {
		// If vault not initialized, run init flow
		if cfg == nil || len(cfg.MasterSalt) == 0 {
			fmt.Println("Vault not initialized. Running setup...")
			runInit()
			// Try again
			cfg, masterKey, err = unlockVault()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	lib := store.NewLibrary(cfg.DataDir, masterKey)
	if err := lib.LoadBooks(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading library: %v\n", err)
		crypto.ZeroBytes(masterKey)
		os.Exit(1)
	}

	// Ensure Inbox exists
	if err := lib.EnsureInbox(); err != nil {
		fmt.Fprintf(os.Stderr, "Error ensuring Inbox: %v\n", err)
	}

	// Build search index
	idx := search.NewIndex()

	// Create AI client (may be nil if no API key)
	var aiClient *ai.Client
	if cfg.GeminiAPIKey != "" {
		aiClient = ai.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	}

	app := ui.NewApp(cfg, lib, idx, aiClient)

	p := tea.NewProgram(app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Handle signals in background
	go func() {
		<-sigCh
		// Graceful shutdown: the Bubble Tea program will handle cleanup
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		crypto.ZeroBytes(masterKey)
		os.Exit(1)
	}

	// Zero key material on exit
	lib.Lock()
	crypto.ZeroBytes(masterKey)
}

