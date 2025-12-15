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

	// ---------------------------------------------------------
	// 1. Firebase Admin SDK Initialization
	// ---------------------------------------------------------
	jsonConfig := os.Getenv("FCM_SERVICE_ACCOUNT_JSON")
	if jsonConfig == "" {
		log.Println("WARNING: FCM_SERVICE_ACCOUNT_JSON not set. Notifications disabled.")
	} else {
		opt := option.WithCredentialsJSON([]byte(jsonConfig))
		fcmApp, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			log.Printf("ERROR: Firebase init failed: %v", err)
		} else {
			fcmClient, err = fcmApp.Messaging(context.Background())
			if err != nil {
				log.Printf("ERROR: FCM client init failed: %v", err)
			} else {
				log.Println("SUCCESS: FCM client initialized.")
			}
		}
	}

	// ---------------------------------------------------------
	// 2. Order Status Change Notification Hook (UPDATED)
	// ---------------------------------------------------------
	app.OnRecordAfterUpdate("orders").Add(func(e *core.RecordUpdateEvent) error {

		oldStatus := e.Record.Original().GetString("status")
		newStatus := e.Record.GetString("status")

		if oldStatus == newStatus {
			return nil
		}

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
			return nil
		}

		customerID := e.Record.GetString("customer")
		if customerID == "" {
			return nil
		}

		// -----------------------------------------------------
		// Find FCM token (NEW DB ACCESS METHOD)
		// -----------------------------------------------------
		tokenRecord, err := e.App.FindFirstRecordByData(
			"fcm_tokens",
			"user",
			customerID,
		)
		if err != nil {
			e.App.Logger().Warn("No FCM token found", "user", customerID)
			return nil
		}

		token := tokenRecord.GetString("token")
		if token == "" {
			return nil
		}

		// -----------------------------------------------------
		// Send Firebase Notification
		// -----------------------------------------------------
		if fcmClient == nil {
			e.App.Logger().Warn("FCM client not initialized")
			return nil
		}

		msg := &messaging.Message{
			Token: token,
			Notification: &messaging.Notification{
				Title: title,
				Body:  body,
			},
		}

		resp, err := fcmClient.Send(context.Background(), msg)
		if err != nil {
			e.App.Logger().Error("FCM send failed", "error", err)
		} else {
			e.App.Logger().Info("Notification sent", "response", resp)
		}

		return nil
	})

	// ---------------------------------------------------------
	// Start PocketBase
	// ---------------------------------------------------------
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
