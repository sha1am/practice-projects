package invite

type AuditLog struct{ Changes []StatusChange }

func NewAuditLog() *AuditLog {
	return &AuditLog{}
}

// AuditLog tracks status changes for invites.
//
// LogChange records a status transition.
// Create a StatusChange and append it to Changes.
func (a *AuditLog) LogChange(inviteID, oldStatus, newStatus string, timestamp int64) {
	// TODO: implement
}

// GetStats returns a map of status -> count for all invites
// belonging to the given assessmentID.
//
// Use store.All() to iterate invites and filter by AssessmentID.
func (a *AuditLog) GetStats(assessmentID string, store *InviteStore) map[string]int {
	// TODO: implement
	return nil
}
