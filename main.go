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

var fcmClient *messaging.Client // Global client for reusability

func main() {
	app := pocketbase.New()

	// 1. Initialize Firebase Admin SDK (FCM v1)
	jsonConfig := os.Getenv("FCM_SERVICE_ACCOUNT_JSON")
	if jsonConfig == "" {
		log.Fatal("FATAL: FCM_SERVICE_ACCOUNT_JSON environment variable is not set. Cannot initialize FCM service.")
	}

	// Use the JSON content for authentication
	opt := option.WithCredentialsJSON([]byte(jsonConfig))
	
	// A simple nil configuration works if the JSON is complete.
	fcmApp, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("FATAL: Error initializing Firebase Admin SDK: %v", err)
	}
	
	// Get the Messaging Client
	fcmClient, err = fcmApp.Messaging(context.Background())
	if err != nil {
		log.Fatalf("FATAL: Error getting FCM client: %v", err)
	}
	log.Println("SUCCESS: FCM v1 client initialized and ready.")


	// 2. PocketBase Hook Logic to send a notification
	// **THIS IS THE FINAL CORRECTION: Using .BindFunc(func) as per documentation.**
	app.OnRecordAfterCreateSuccess("push_tokens").BindFunc(func(e *core.RecordEvent) error { 
		// Get the device token from the newly created record
		token := e.Record.GetString("token") 
		
		if token == "" {
			e.App.Logger().Warn("push_tokens record created without a device token. Skipping notification.", "recordId", e.Record.Id)
			return nil
		}

		// Define the FCM v1 message payload
		message := &messaging.Message{
			Notification: &messaging.Notification{
				Title: "Welcome!",
				Body:  "Your device is successfully registered for notifications.",
			},
			Token: token, // The target device token
			// Data: map[string]string{ "notification_type": "welcome", "user_id": e.Record.GetString("user") }, 
		}

		// Send the message using the global client
		response, err := fcmClient.Send(context.Background(), message)
		if err != nil {
			// Log the error but return nil so the DB transaction remains successful
			e.App.Logger().Error("Failed to send FCM v1 message", "error", err.Error(), "token", token)
			return nil 
		}

		e.App.Logger().Info("Successfully sent FCM v1 message", "token", token, "response", response)
		
		// The documentation mentions calling e.Next() if we want to proceed with the execution chain.
		// Since we only want to send a notification and don't need to block or modify the chain, 
		// returning nil (success) is usually sufficient for simple handlers like this one.
		return nil
	}) 

	// Start the PocketBase application
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}