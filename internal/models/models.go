package models

import "time"

type Organization struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	TelegramChatID  *int64    `json:"telegram_chat_id"`
	GoogleCalendarID *string  `json:"google_calendar_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type User struct {
	ID             int       `json:"id"`
	OrganizationID int       `json:"organization_id"`
	TelegramID     *int64    `json:"telegram_id"`
	Username       *string   `json:"username"`
	Email          *string   `json:"email"`
	PasswordHash   *string   `json:"-"`
	Role           string    `json:"role"`
	FullName       *string   `json:"full_name"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Ticket struct {
	ID              int       `json:"id"`
	OrganizationID  int       `json:"organization_id"`
	CustomerID      *int      `json:"customer_id"`
	AssignedAgentID *int      `json:"assigned_agent_id"`
	Title           string    `json:"title"`
	Description     *string   `json:"description"`
	Status          string    `json:"status"`
	Priority        string    `json:"priority"`
	TelegramMessageID *int    `json:"telegram_message_id"`
	TelegramChatID  *int64    `json:"telegram_chat_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Message struct {
	ID               int       `json:"id"`
	TicketID         int       `json:"ticket_id"`
	UserID           *int      `json:"user_id"`
	Content          string    `json:"content"`
	TelegramMessageID *int     `json:"telegram_message_id"`
	IsFromCustomer   bool      `json:"is_from_customer"`
	CreatedAt        time.Time `json:"created_at"`
}

type Attachment struct {
	ID         int       `json:"id"`
	MessageID  int       `json:"message_id"`
	FileName   string    `json:"file_name"`
	FilePath   string    `json:"file_path"`
	FileSize   *int64    `json:"file_size"`
	MimeType   *string   `json:"mime_type"`
	CreatedAt  time.Time `json:"created_at"`
}

type GoogleCalendarToken struct {
	ID             int       `json:"id"`
	OrganizationID int       `json:"organization_id"`
	AccessToken    string    `json:"-"`
	RefreshToken   *string   `json:"-"`
	TokenType      *string   `json:"token_type"`
	Expiry         *time.Time `json:"expiry"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
