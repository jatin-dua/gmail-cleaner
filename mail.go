package main

import (
	"fmt"
	"log"
	"regexp"
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

func startMailDeletion(srv *gmail.Service, sender string) error {
	MAX_RESULT_SIZE := "500"
	nextPageToken := ""
	deleteCnt := 0
	processedCnt := 0
	var messagesToDelete []string

	months := 1
	fmt.Print("Specify duration in months (Default: 1, i.e, delete from current date to previous month): ")
	fmt.Scanf("%d", &months)

	// Define regex to capture "2 Jan 2006" format (day, month, year)
	datePattern := `\d{1,2} (Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d{4}`
	dateRegex := regexp.MustCompile(datePattern)

	deleteTillDate := time.Now().AddDate(0, -months, 0)
	fmt.Printf("Delete till Date: %v\n\n", deleteTillDate)

	const layout = "2 Jan 2006"

	for ok := true; ok; ok = nextPageToken != "" {
		messageIds, pageToken, err := listMessageIds(srv, nextPageToken, MAX_RESULT_SIZE)
		if err != nil {
			// log.Fatalf("Unable to retrieve message Ids: %v", err)
			return err
		}
		nextPageToken = pageToken
		for _, id := range messageIds {
			message, err := getMessage(srv, id)
			if err != nil {
				return err
			}
			processedCnt++

			match := dateRegex.FindString(message.Date)
			if match == "" {
				fmt.Println("No matching date found")
				return err
			}

			parsedDate, err := time.Parse(layout, match)
			if err != nil {
				fmt.Println("Error parsing date:", err)
				return err
			}

			if parsedDate.Month() < deleteTillDate.Month() {
				nextPageToken = ""
				break
			}

			if senderIsTarget(message.From, sender) {
				fmt.Printf("Id: %s\nFrom: %s\nDate: %s\nSubject: %s\n\n",
					message.Id,
					message.From,
					message.Date,
					message.Subject,
				)
				deleteCnt++
				messagesToDelete = append(messagesToDelete, message.Id)
			}
			// To avoid rate-limiting
			time.Sleep(25 * time.Millisecond)
		}

		if len(messagesToDelete) == 0 {
			log.Println("No messages found from the target")
			return nil
		}
	}

	fmt.Println()
	fmt.Printf("Processed %d messages\n", processedCnt)
	fmt.Printf("Proceed deleting %d messages [y/N]:", deleteCnt)

	var choice string
	fmt.Scanln(&choice)
	if strings.ToLower(choice) == "y" {
		err := deleteMessages(srv, messagesToDelete)
		if err != nil {
			// log.Fatalf("Failed to delete messages: %v", err)
			return err
		}
	}

	return nil
}

func listMessageIds(srv *gmail.Service, nextPageToken, maxResults string) ([]string, string, error) {
	var messageIds []string
	user := "me"

	callOptions := []googleapi.CallOption{
		googleapi.QueryParameter("maxResults", maxResults),
	}

	if nextPageToken != "" {
		callOptions = append(callOptions, googleapi.QueryParameter("pageToken", nextPageToken))
	}

	r, err := srv.Users.Messages.List(user).Do(callOptions...)
	if err != nil {
		return nil, "", err
	}
	if len(r.Messages) == 0 {
		fmt.Println("No Messages found.")
		return nil, "", err
	}

	for _, l := range r.Messages {
		messageIds = append(messageIds, l.Id)
	}

	return messageIds, r.NextPageToken, nil
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

		if (message.From != "") && (message.Subject != "") && (message.Date != "") {
			break
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
