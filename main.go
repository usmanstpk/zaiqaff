app.OnRecordAfterUpdate("orders").BindFunc(func(e *core.RecordUpdateEvent) error {

	// DEBUG LOG 1: Confirms the hook is firing for an order update
	e.App.Logger().Info("--- HOOK FIRED: Order Record Updated ---", "recordId", e.Record.Id, "collection", e.Record.Collection().Name)

	oldStatus := e.Record.Original().GetString("status")
	newStatus := e.Record.GetString("status")

	// If status hasn't changed, do nothing
	if oldStatus == newStatus {
		e.App.Logger().Info("Hook Exited: Status field did not change.", "old", oldStatus, "new", newStatus)
		return nil // Exits silently if only other fields (like total) changed
	}

	// DEBUG LOG 2: Confirms a status change was detected
	e.App.Logger().Info("Status Change Detected! Sending Notification...", "old", oldStatus, "new", newStatus)

	// ... (rest of your notification logic) ...
	// The rest of your code is here, which handles status specific logic and token lookup

	// ... (Your notification sending logic) ...

	return nil
})