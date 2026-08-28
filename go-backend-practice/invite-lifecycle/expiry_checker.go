package invite

type ExpiryChecker struct {
	store    *InviteStore
	auditLog *AuditLog
}

func NewExpiryChecker(store *InviteStore, auditLog *AuditLog) *ExpiryChecker {
	return &ExpiryChecker{store: store, auditLog: auditLog}
}

// TODO: expire PENDING invites with ExpiresAt <= now.
func (e *ExpiryChecker) Run(now int64) {
}
