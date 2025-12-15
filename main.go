package main

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	// Hook: after a record in "orders" collection is updated
	app.OnRecordAfterUpdate("orders", func(e *core.RecordEvent) error {
		newStatus := e.Record.Get("status")
		fmt.Printf("[LOG] Order updated: ID=%s, New status=%v\n", e.Record.Id, newStatus)
		return nil
	})

	// Start PocketBase
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
