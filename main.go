package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "path/filepath"

    "github.com/pocketbase/dbx"
    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"
    "github.com/pocketbase/pocketbase/models" // CRITICAL: Need this import for Model interface
    
    firebase "firebase.google.com/go"
    "firebase.google.com/go/messaging"
    "google.golang.org/api/option"
)

// Global Firebase client to be initialized once
var fcmClient *messaging.Client

func main() {
    app := pocketbase.New()

    // 1. Initialize Firebase Client
    // FIX 1: Replaced .Add() with the modern .BindFunc()
    app.OnBeforeServe().BindFunc(func(e *core.ServeEvent) error {
        // Get service account JSON from environment variable
        saJSON := os.Getenv("FCM_SERVICE_ACCOUNT_JSON")
        if saJSON == "" {
            e.App.Logger().Info("FCM_SERVICE_ACCOUNT_JSON is not set. Skipping Firebase initialization.")
            return nil
        }

        // Use the JSON credentials string to initialize the app
        opt := option.WithCredentialsJSON([]byte(saJSON))
        firebaseApp, err := firebase.NewApp(context.Background(), nil, opt)
        if err != nil {
            e.App.Logger().Error("FAILED: Firebase initialization failed.", "error", err)
            return nil
        }

        // Create the FCM client
        fcmClient, err = firebaseApp.Messaging(context.Background())
        if err != nil {
            e.App.Logger().Error("FAILED: FCM client creation failed.", "error", err)
            return nil
        }

        e.App.Logger().Info("SUCCESS: FCM client initialized.")
        return nil
    })

    // 2. Register the OnModelAfterUpdate hook for the 'orders' collection
    // FIX 2: Replaced app.OnRecordAfterUpdate with app.OnModelAfterUpdate
    // FIX 3: Updated function signature from *core.RecordUpdateEvent (deprecated) to *core.ModelEvent
    app.OnModelAfterUpdate("orders").BindFunc(func(e *core.ModelEvent) error {
        // We expect the model to be a *models.Record for the "orders" collection
        record, ok := e.Model.(*models.Record)
        if !ok {
            // Should not happen for an "orders" collection hook, but good practice
            e.App.Logger().Warn("Hook Exited: Event model is not a Record.", "modelType", e.Model.TableName())
            return nil
        }
        
        // --- START DEBUG LOGGING ---
        e.App.Logger().Info("--- HOOK FIRED: Order Record Updated ---", "recordId", record.Id, "collection", record.Collection().Name)
        // --- END DEBUG LOGGING ---

        // Ensure the FCM client is ready
        if fcmClient == nil {
            e.App.Logger().Warn("Hook Exited: FCM client is not initialized.")
            return nil
        }

        // PocketBase v0.34.2 (and later) provides OriginalCopy() to access old data
        oldRecord := record.OriginalCopy()
        
        // Ensure oldRecord is not nil before attempting to access fields
        if oldRecord == nil {
            e.App.Logger().Warn("Hook Exited: Could not retrieve original record copy.", "recordId", record.Id)
            return nil
        }

        oldStatus := oldRecord.GetString("status")
        newStatus := record.GetString("status") // The main 'record' has the new data

        // If status hasn't changed, exit silently
        if oldStatus == newStatus {
            e.App.Logger().Info("Hook Exited: Status field did not change.", "old", oldStatus, "new", newStatus)
            return nil
        }

        // --- START DEBUG LOGGING ---
        e.App.Logger().Info("Status Change Detected! Sending Notification...", "old", oldStatus, "new", newStatus)
        // --- END DEBUG LOGGING ---

        // 2.1 Get the customer ID
        customerID := record.GetString("customer")
        if customerID == "" {
            e.App.Logger().Warn("Hook Exited: No customer ID found for order.", "orderId", record.Id)
            return nil
        }

        // 2.2 Lookup the FCM token for the customer
        tokenRecord, err := e.Dao.FindFirstRecordByData("fcm_tokens", dbx.Params{"user": customerID})
        if err != nil {
            e.App.Logger().Info("Hook Exited: No FCM token found for customer.", "customerID", customerID, "error", err)
            return nil
        }

        fcmToken := tokenRecord.GetString("token")
        if fcmToken == "" {
            e.App.Logger().Warn("Hook Exited: Token record found but token field is empty.", "customerID", customerID)
            return nil
        }

        // 2.3 Determine notification message based on the new status
        var title, body string
        switch newStatus {
        case "cooking":
            title = "Order Update! 🍳"
            body = "Your order is now being prepared by the kitchen."
        case "out_for_delivery":
            title = "It's on the way! 🛵"
            body = "Your order is out for delivery and will arrive soon."
        case "completed":
            title = "Order Delivered! ✅"
            body = "Your order has been successfully delivered. Enjoy!"
        case "cancelled":
            title = "Order Cancelled 🚫"
            body = "Your order has been cancelled."
        default:
            // No notification for 'pending' or other statuses
            e.App.Logger().Info("Hook Exited: Status change does not require a notification.", "newStatus", newStatus)
            return nil
        }

        // 2.4 Send the FCM message
        message := &messaging.Message{
            Token: fcmToken,
            Notification: &messaging.Notification{
                Title: title,
                Body:  body,
            },
            Data: map[string]string{
                "orderId": record.Id,
                "status":  newStatus,
            },
        }

        response, err := fcmClient.Send(context.Background(), message)
        if err != nil {
            e.App.Logger().Error("FCM send failed.", "error", err, "customerID", customerID, "status", newStatus)
            return nil
        }

        e.App.Logger().Info("🚀 Notification sent successfully.", "response", response, "status", newStatus, "customerID", customerID)

        return nil
    })

    // 3. Start the PocketBase application
    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}

// NOTE: This utility function is included in case you need to
// embed the FCM JSON file later, but using an environment variable is cleaner.
func embedFCMServiceAccountFile(app *pocketbase.PocketBase) string {
    // Assumes the file is named service-account.json and is in the same directory
    saFile := "service-account.json"
    saPath := filepath.Join(os.Getenv("POCKETBASE_DIR"), saFile)

    // Read from the environment variable provided by Coolify/Docker
    saJSON := os.Getenv("FCM_SERVICE_ACCOUNT_JSON")

    // If the environment variable is set, use it directly as the content
    if saJSON != "" {
        // Validate JSON content (optional but helpful)
        var j map[string]interface{}
        if json.Unmarshal([]byte(saJSON), &j) == nil {
            return saJSON // Return the valid JSON string content
        }
    }

    // Fallback to reading a file if the environment variable is empty or invalid
    data, err := os.ReadFile(saPath)
    if err != nil {
        app.Logger().Warn("Could not find FCM service-account.json file or use FCM_SERVICE_ACCOUNT_JSON env var.")
        return ""
    }
    return string(data)
}