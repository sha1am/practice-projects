package invite

type BulkProcessor struct{ service *InviteService }

func NewBulkProcessor(service *InviteService) *BulkProcessor { return &BulkProcessor{service: service} }

// TODO: process unique emails and return Sent, Skipped and Failed.
func (b *BulkProcessor) BulkSend(assessmentID string, emails []string, expiresAt, now int64) BulkSendResult {
	return BulkSendResult{}
}
