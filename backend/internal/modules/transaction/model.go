package transaction

type Task struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Project   string   `json:"project"`
	Priority  string   `json:"priority"`
	DueDate   string   `json:"due_date"`
	Completed bool     `json:"completed"`
	Tags      []string `json:"tags"`
}

func (t Task) GetID() string    { return t.ID }
func (t *Task) SetID(id string) { t.ID = id }

// Project tetap dipertahankan dari struktur lama. Nanti bisa dipisahkan
// kembali kalau domain transaction sudah ditulis ulang.
type Project struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DueDate  string `json:"due_date"`
	Color    string `json:"color"`
	Icon     string `json:"icon"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
}

func (p Project) GetID() string    { return p.ID }
func (p *Project) SetID(id string) { p.ID = id }
