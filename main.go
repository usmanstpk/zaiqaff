package main

import (
	"context"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	// Official Firebase Admin SDK imports
	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

var fcmClient *messaging.Client // Global client

func main() {
	app := pocketbase.New()

	// -----------------------------------------------------------------
	// 1. Initialize Firebase Admin SDK
	// -----------------------------------------------------------------
	jsonConfig := os.Getenv("FCM_SERVICE_ACCOUNT_JSON")
	if jsonConfig == "" {
		// Log warning but don't crash, so the app can still run without notifications if needed
		log.Println("WARNING: FCM_SERVICE_ACCOUNT_JSON is not set. Notifications will not work.")
	} else {
		opt := option.WithCredentialsJSON([]byte(jsonConfig))
		fcmApp, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			log.Printf("ERROR: Failed to init Firebase App: %v", err)
		} else {
			// Get the Messaging Client
			fcmClient, err = fcmApp.Messaging(context.Background())
			if err != nil {
				log.Printf("ERROR: Failed to get FCM client: %v", err)
			} else {
				log.Println("SUCCESS: FCM client initialized and ready.")
			}
		}
	}

	// -----------------------------------------------------------------
	// 2. HOOK: Send Notification when Order Status Changes
	// -----------------------------------------------------------------
	app.OnRecordAfterUpdateRequest("orders").BindFunc(func(e *core.RecordEvent) error {
		// A. Get the old and new status
		oldStatus := e.Record.Original().GetString("status")
		newStatus := e.Record.GetString("status")

		// If status hasn't changed, do nothing
		if oldStatus == newStatus {
			return nil
		}

		// B. Determine the message based on the new status
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
			return nil // Unknown status, skip
		}

		// C. Get the Customer ID from the order
		customerID := e.Record.GetString("customer")
		if customerID == "" {
			return nil
		}

		// D. Find the FCM Token for this customer
		// Note: We use "fcm_tokens" (from your schema), not "push_tokens"
		tokenRecord, err := app.Dao().FindFirstRecordByData("fcm_tokens", "user", customerID)
		if err != nil {
			// No token found for this user
			log.Printf("No FCM token found for user %s", customerID)
			return nil
		}

		token := tokenRecord.GetString("token")
		if token == "" {
			return nil
		}

		// E. Send the Notification (Real Firebase Send)
		if fcmClient != nil {
			message := &messaging.Message{
				Notification: &messaging.Notification{
					Title: title,
					Body:  body,
				},
				Token: token,
			}

			response, err := fcmClient.Send(context.Background(), message)
			if err != nil {
				e.App.Logger().Error("Failed to send FCM message", "error", err, "token", token)
			} else {
				e.App.Logger().Info("🚀 Notification Sent!", "response", response, "status", newStatus)
			}
		} else {
			log.Println("Skipping notification: FCM client not initialized.")
		}

		return e.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
