package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
)

type Message struct {
	Id      string
	From    string
	Subject string
	Date    string
}

func startMailer(srv *gmail.Service, sender string) error {
	MAX_RESULT_SIZE := "500"
	messageIds, err := listMessageIds(srv, MAX_RESULT_SIZE)
	if err != nil {
		// log.Fatalf("Unable to retrieve message Ids: %v", err)
		return err
	}

	var messagesToDelete []string
	for _, id := range messageIds {
		message, err := getMessage(srv, id)
		if err != nil {
			return err
		}

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
			// log.Fatalf("Failed to delete messages: %v", err)
			return err
		}
	}

	return nil
}

func listMessageIds(srv *gmail.Service, maxResults string) ([]string, error) {
	var messageIds []string
	user := "me"
	r, err := srv.Users.Messages.List(user).Do(googleapi.QueryParameter("maxResults", maxResults))
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

func getMessage(srv *gmail.Service, id string) (Message, error) {
	var message Message
	message.Id = id

	userId := "me"
	r, err := srv.Users.Messages.Get(userId, id).Do()
	if err != nil {
		// log.Fatalf("Unable to retrieve message: %v", err)
		return Message{}, err
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

	return message, nil
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
