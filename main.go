package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type Config struct {
	srv           *gmail.Service
	sender        string
	deleteAfter   time.Time
	maxResultSize string
	deleteLimit   int
}

func startServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
			return
		}
		instruction := fmt.Sprintf("Paste this code into the terminal and press Enter to complete OAuth\n%s\n\nYou can now safely close this tab.", code)
		fmt.Fprintf(w, instruction)
	})
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	go startServer()

	var sender string
	var deleteAfter string
	var limit int

	flag.StringVar(&sender, "sender", "", "Target to delete messages")
	flag.StringVar(&deleteAfter, "after", "", "Delete mails after this date")
	flag.IntVar(&limit, "limit", 100, "Limit deletion to a specific number")
	flag.Parse()

	if sender == "" {
		log.Fatalf("usage: ./gc -sender=<target> -after=<dd/mm/yyyy>")
	}

	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.MailGoogleComScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	const dateLayout = "2/1/2006"

	parsedDeleteAfter, err := time.Parse(dateLayout, deleteAfter)
	if err != nil {
		log.Fatalf("Error parsing date: %v", err)
	}

	cfg := &Config{
		srv:           srv,
		sender:        sender,
		deleteAfter:   parsedDeleteAfter,
		maxResultSize: "100",
		deleteLimit:   limit,
	}

	if err := startMailDeletion(cfg); err != nil {
		log.Fatalf("Error occurred in mailer: %v", err)
	}
}
