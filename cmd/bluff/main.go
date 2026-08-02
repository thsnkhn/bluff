// Command bluff is the terminal client for Bluff.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/thsnkhn/bluff/internal/api"
	"github.com/thsnkhn/bluff/internal/credentials"
	"github.com/thsnkhn/bluff/internal/ui"
)

var (
	version = "dev"
	commit  = "none"
	builtAt = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "print version information")
	flag.Parse()
	if *showVersion {
		fmt.Printf("bluff %s (%s, %s)\n", version, commit, builtAt)
		return
	}

	baseURL := os.Getenv("BLUFF_API_URL")
	if baseURL == "" {
		baseURL = "https://api.bluff.thsnkhn.com"
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	client, err := api.NewClient(baseURL, httpClient)
	if err != nil {
		log.Fatalf("configure API: %v", err)
	}

	store := credentials.NewKeyringStore(baseURL)
	model := ui.New(client, store, ui.BuildInfo{Version: version})
	program := tea.NewProgram(model, tea.WithContext(context.Background()))
	if _, err := program.Run(); err != nil {
		log.Fatalf("run bluff: %v", err)
	}
}
