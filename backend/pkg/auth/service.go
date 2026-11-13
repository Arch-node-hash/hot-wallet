package auth

// User adalah model user skeleton.
// TODO: sesuaikan field dengan kebutuhan aplikasi.
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (u User) GetID() string    { return u.ID }
func (u *User) SetID(id string) { u.ID = id }

// Service adalah skeleton service Auth.
// TODO: tambahkan dependency (repo/store) dan implementasikan method-nya.
type Service struct{}

func NewService() *Service { return &Service{} }
