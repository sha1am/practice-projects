package invite

type ExpiryChecker struct {
	store    *InviteStore
	auditLog *AuditLog
}

func NewExpiryChecker(store *InviteStore, auditLog *AuditLog) *ExpiryChecker {
	return &ExpiryChecker{store: store, auditLog: auditLog}
}

// Run expires all PENDING invites whose ExpiresAt is less than or equal to now.
//
// For every matching invite:
//   - Set status to "EXPIRED".
//   - Save the invite.
//   - Log the status change.
//
// SENT and already EXPIRED invites must not be changed.
func (e *ExpiryChecker) Run(now int64) {
	// TODO: implement
}
