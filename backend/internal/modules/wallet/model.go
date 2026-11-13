package wallet

import (
	"hotwallet/backend/internal/modules/activity"
	"hotwallet/backend/internal/modules/contact"
	"hotwallet/backend/internal/modules/portfolio"
	"hotwallet/backend/internal/modules/transaction"
)

type WeeklyActivity struct {
	Day   string `json:"day"`
	Value int    `json:"value"`
	Label string `json:"label"`
}

type Progress struct {
	Percent    int `json:"percent"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Pending    int `json:"pending"`
}

// RawData adalah struktur data wallet dari koleksi "wallet".
type RawData struct {
	Stats          []portfolio.Stat `json:"stats"`
	WeeklyActivity []WeeklyActivity `json:"weekly_activity"`
	Progress       Progress         `json:"progress"`
}

// User adalah data user minimal untuk wallet
type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	Initials string `json:"initials"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

func (u User) GetID() string    { return u.ID }
func (u *User) SetID(id string) { u.ID = id }

// TimeEntry tetap dipertahankan dari struktur lama.
type TimeEntry struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	TaskID    string `json:"task_id"`
	Seconds   int    `json:"seconds"`
	Running   bool   `json:"running"`
	StartedAt string `json:"started_at"`
}

func (e TimeEntry) GetID() string    { return e.ID }
func (e *TimeEntry) SetID(id string) { e.ID = id }

// Response adalah agregasi data wallet.
type Response struct {
	User           User                  `json:"user"`
	Stats          []portfolio.Stat      `json:"stats"`
	WeeklyActivity []WeeklyActivity      `json:"weekly_activity"`
	Projects       []transaction.Project `json:"projects"`
	Team           []contact.Member      `json:"team"`
	Reminders      []activity.Reminder   `json:"reminders"`
	Progress       Progress              `json:"progress"`
	TimeEntry      TimeEntry             `json:"time_entry"`
}
