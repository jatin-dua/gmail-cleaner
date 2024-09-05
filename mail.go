package main

import (
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
)

type Message struct {
	Id      string
	From    string
	Subject string
	Date    string
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
