package invite

type AuditLog struct{ Changes []StatusChange }

func NewAuditLog() *AuditLog { return &AuditLog{} }

// TODO: append a StatusChange.
func (a *AuditLog) LogChange(inviteID, oldStatus, newStatus string, timestamp int64) {
}

// TODO: count statuses for the supplied assessment.
func (a *AuditLog) GetStats(assessmentID string, store *InviteStore) map[string]int {
	return nil
}
