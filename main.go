package main

import (
	"context"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

var fcmClient *messaging.Client

func main() {
	app := pocketbase.New()

	// ----------------------------------------
	// 1. Firebase initialization
	// ----------------------------------------
	jsonConfig := os.Getenv("FCM_SERVICE_ACCOUNT_JSON")
	if jsonConfig == "" {
		log.Println("WARNING: FCM_SERVICE_ACCOUNT_JSON not set — FCM disabled.")
	} else {
		opt := option.WithCredentialsJSON([]byte(jsonConfig))
		fcmApp, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			log.Printf("Firebase init error: %v", err)
		} else {
			fcmClient, err = fcmApp.Messaging(context.Background())
			if err != nil {
				log.Printf("FCM client error: %v", err)
			} else {
				log.Println("SUCCESS: FCM client initialized.")
			}
		}
	}

	// ----------------------------------------
	// 2. Hook: Order status change
	// ----------------------------------------
	app.OnRecordUpdateRequest("orders").BindFunc(func(e *core.RecordRequestEvent) error {
		// 1) Old vs new status
		oldStatus := e.Record.Original().GetString("status")
		newStatus := e.Record.GetString("status")

		if oldStatus == newStatus {
			// no change → just continue
			return e.Next()
		}

		// 2) Notification message
		var title, body string
		switch newStatus {
		case "cooking":
			title = "Order Update"
			body = "Your food is being prepared! 🍳"
		case "out_for_delivery":
			title = "Order Update"
			body = "Your rider is on the way! 🛵"
		case "completed":
			title = "Delivered"
			body = "Enjoy your meal! 😋"
		case "cancelled":
			title = "Order Cancelled"
			body = "We are sorry, your order was cancelled."
		default:
			return e.Next()
		}

		// 3) Get customer ID from order
		customerID := e.Record.GetString("customer")
		if customerID == "" {
			return e.Next()
		}

		// 4) Find FCM token for this customer
		tokenRecord, err := e.App.FindFirstRecordByData(
			"fcm_tokens",
			"user",
			customerID,
		)
		if err != nil {
			log.Printf("No FCM token found for customer %s", customerID)
			return e.Next()
		}

		token := tokenRecord.GetString("token")
		if token == "" {
			return e.Next()
		}

		// 5) Send FCM notification
		if fcmClient != nil {
			msg := &messaging.Message{
				Token: token,
				Notification: &messaging.Notification{
					Title: title,
					Body:  body,
				},
			}
			_, err := fcmClient.Send(context.Background(), msg)
			if err != nil {
				log.Printf("FCM send failed: %v", err)
			} else {
				log.Printf("Notification sent for status %s", newStatus)
			}
		}

		return e.Next()
	})

	// ----------------------------------------
	// Start PocketBase server
	// ----------------------------------------
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
