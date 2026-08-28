package invite

type Invite struct {
	InviteID       string
	AssessmentID   string
	CandidateEmail string
	Status         string
	ExpiresAt      int64
	SentAt         *int64
}

type StatusChange struct {
	InviteID  string
	OldStatus string
	NewStatus string
	Timestamp int64
}

type BulkSendResult struct {
	Sent    []string
	Skipped []string
	Failed  []string
}
