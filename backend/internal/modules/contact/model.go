package contact

type Member struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Avatar      string `json:"avatar"`
	Initials    string `json:"initials"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	ActiveTasks int    `json:"active_tasks"`
	CurrentTask string `json:"current_task"`
	TaskStatus  string `json:"task_status"`
}

func (m Member) GetID() string    { return m.ID }
func (m *Member) SetID(id string) { m.ID = id }
