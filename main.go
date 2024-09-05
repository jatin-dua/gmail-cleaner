package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func main() {
	var sender string
	flag.StringVar(&sender, "sender", "", "Target to delete messages")
	flag.Parse()

	if sender == "" {
		log.Fatalf("usage: mailer -sender <target>")
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

	MAX_RESULT_SIZE := "500"
	messageIds, err := listMessageIds(srv, MAX_RESULT_SIZE)
	if err != nil {
		log.Fatalf("Unable to retrieve message Ids: %v", err)
	}

	var messagesToDelete []string
	for _, id := range messageIds {
		message := getMessage(srv, id)
		if senderIsTarget(message.From, sender) {
			fmt.Printf("[DELETE]\nId: %s\nFrom: %s\nDate: %s\nSubject: %s\n\n", message.Id, message.From, message.Date, message.Subject)
			messagesToDelete = append(messagesToDelete, message.Id)
		}
		// To avoid rate-limiting
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("Proceed deleting %d messages: [y/N] ", len(messagesToDelete))
	var choice string
	fmt.Scanln(&choice)
	if strings.ToLower(choice) == "y" {
		err = deleteMessages(srv, messagesToDelete)
		if err != nil {
			log.Fatalf("Failed to delete messages: %v", err)
		}
	}
}
