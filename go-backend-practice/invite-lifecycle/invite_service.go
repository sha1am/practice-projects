package invite

import "github.com/google/uuid"

type InviteService struct {
	store    *InviteStore
	auditLog *AuditLog
}

func NewInviteService(store *InviteStore, auditLog *AuditLog) *InviteService {
	return &InviteService{store: store, auditLog: auditLog}
}

// CreateInvite creates a new invite.
// - Reject active duplicates for the same assessment and candidate.
// - A duplicate is an invite with status PENDING or SENT.
// - New invites must start with status PENDING.
// - Store the invite before returning it.
func (s *InviteService) CreateInvite(assessmentID, candidateEmail string, expiresAt int64) *Invite {
	// TODO: implement
	_ = uuid.NewString()
	return nil
}

// SendInvite transitions an invite from PENDING to SENT (or EXPIRED).
//
// Return values by case:
//   - Invite not found:                 "NOT_FOUND"
//   - Already "SENT":                   "ALREADY_SENT"
//   - Already "EXPIRED":                "ALREADY_EXPIRED"
//   - PENDING and expiresAt <= now:     set "EXPIRED", save, log change, return "EXPIRED"
//   - PENDING and expiresAt > now:      set "SENT", set sentAt=now, save, log change, return "SENT"
func (s *InviteService) SendInvite(inviteID string, now int64) string {
	// TODO: implement
	return ""
}

// Resend resets an EXPIRED invite back to PENDING so it can be sent again.
//
// Return values by case:
//   - Invite not found:                 "NOT_FOUND"
//   - Not "EXPIRED":                    "CANNOT_RESEND"
//   - Otherwise: set status "PENDING", update expiresAt to newExpiresAt,
//     clear sentAt (nil), save, log change, return "RESENT"
func (s *InviteService) Resend(inviteID string, newExpiresAt, now int64) string {
	// TODO: implement
	return ""
}
