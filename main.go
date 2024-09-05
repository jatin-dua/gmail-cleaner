package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

type Message struct {
	Id      string
	From    string
	Subject string
	Date    string
}

func listMessageIds(srv *gmail.Service) ([]string, error) {
	var messageIds []string
	user := "me"
	r, err := srv.Users.Messages.List(user).Do()
	if err != nil {
		return nil, err
	}
	if len(r.Messages) == 0 {
		fmt.Println("No Messages found.")
		return nil, err
	}
	// fmt.Printf("NextPageToken: %s\n", r.NextPageToken)
	for _, l := range r.Messages {
		messageIds = append(messageIds, l.Id)
	}
	return messageIds, nil
}

func getMessage(srv *gmail.Service, id string) Message {
	var message Message
	message.Id = id

	userId := "me"
	r, err := srv.Users.Messages.Get(userId, id).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve message: %v", err)
	}
	for _, header := range r.Payload.Headers {
		switch strings.ToLower(header.Name) {
		case "from":
			message.From = header.Value
		case "subject":
			message.Subject = header.Value
		case "date":
			message.Date = header.Value
		}
	}
	return message
}

func senderIsTarget(from, target string) bool {
	return strings.Contains(from, target)
}

func deleteMessages(srv *gmail.Service, messagesToDelete []string) error {
	log.Println("Messages deletion in Progress")
	err := srv.Users.Messages.BatchDelete("me", &gmail.BatchDeleteMessagesRequest{
		Ids: messagesToDelete,
	}).Do()
	if err != nil {
		return err
	}
	log.Println("Messages deletion succeeded")

	return nil
}

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

	messageIds, err := listMessageIds(srv)
	if err != nil {
		log.Fatal("Unable to retrieve message Ids: %v", err)
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

	fmt.Print("Continue Delete: [y/N] ")
	var choice string
	fmt.Scanln(&choice)
	if choice == "y" {
		err = deleteMessages(srv, messagesToDelete)
		if err != nil {
			log.Fatal("Failed to delete messages: %v", err)
		}
	}
}
