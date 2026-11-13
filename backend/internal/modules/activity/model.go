package activity

type Event struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Date     string `json:"date"`
	Time     string `json:"time"`
	Duration string `json:"duration"`
	Type     string `json:"type"`
	Color    string `json:"color"`
}

func (e Event) GetID() string    { return e.ID }
func (e *Event) SetID(id string) { e.ID = id }

// Reminder tetap dipertahankan dari struktur lama.
type Reminder struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Date       string `json:"date"`
	Time       string `json:"time"`
	MeetingURL string `json:"meeting_url,omitempty"`
}

func (r Reminder) GetID() string    { return r.ID }
func (r *Reminder) SetID(id string) { r.ID = id }
