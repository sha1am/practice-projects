package invite

type InviteStore struct{ data map[string]*Invite }

func NewInviteStore() *InviteStore           { return &InviteStore{data: make(map[string]*Invite)} }
func (s *InviteStore) Save(inv *Invite)      { s.data[inv.InviteID] = inv }
func (s *InviteStore) Get(id string) *Invite { return s.data[id] }
func (s *InviteStore) All() []*Invite {
	out := make([]*Invite, 0, len(s.data))
	for _, v := range s.data {
		out = append(out, v)
	}
	return out
}
func (s *InviteStore) GetByStatus(status string) []*Invite {
	var out []*Invite
	for _, v := range s.data {
		if v.Status == status {
			out = append(out, v)
		}
	}
	return out
}
