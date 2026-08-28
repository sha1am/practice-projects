package invite

import "github.com/google/uuid"

type InviteService struct {
	store    *InviteStore
	auditLog *AuditLog
}

func NewInviteService(store *InviteStore, auditLog *AuditLog) *InviteService {
	return &InviteService{store: store, auditLog: auditLog}
}

// TODO: reject active duplicates and create a PENDING invite.
func (s *InviteService) CreateInvite(assessmentID, candidateEmail string, expiresAt int64) *Invite {
	_ = uuid.NewString()
	return nil
}

// TODO: implement PENDING -> SENT / EXPIRED lifecycle.
func (s *InviteService) SendInvite(inviteID string, now int64) string {
	return ""
}

// TODO: only EXPIRED invites can be resent.
func (s *InviteService) Resend(inviteID string, newExpiresAt, now int64) string {
	return ""
}
