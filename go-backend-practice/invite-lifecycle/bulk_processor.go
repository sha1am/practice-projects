package invite

type BulkProcessor struct {
	service *InviteService
}

func NewBulkProcessor(service *InviteService) *BulkProcessor {
	return &BulkProcessor{service: service}
}

// BulkSend creates and sends invites for a list of emails.
//
// For each email:
//   - Call CreateInvite.
//   - If nil (conflict/duplicate), append the email to Skipped.
//   - Otherwise call SendInvite on the new invite.
//   - If "SENT", append invite ID to Sent.
//   - If "EXPIRED", append invite ID to Failed.
//
// Process each email only once; duplicate emails in the input are Skipped.
//
// Return a BulkSendResult with Sent, Skipped, and Failed slices.
func (b *BulkProcessor) BulkSend(assessmentID string, emails []string, expiresAt, now int64) BulkSendResult {
	// TODO: implement
	return BulkSendResult{}
}
