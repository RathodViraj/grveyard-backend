package users

import "time"

type User struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	Role          string    `json:"role"`
	ProfilePicURL string    `json:"profile_pic_url"`
	UUID          string    `json:"uuid"`
	CreatedAt     time.Time `json:"created_at"`
}

type UserList struct {
	Items []User `json:"items"`
	Total int64  `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}
