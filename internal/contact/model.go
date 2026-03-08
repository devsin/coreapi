package contact

import "time"

// MessageDTO is the API response for a contact message.
type MessageDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Subject   string    `json:"subject"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateContactMessageRequest is the incoming payload for POST /api/contact.
type CreateContactMessageRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

// Validation constants.
const (
	maxNameLength    = 100
	maxEmailLength   = 255
	maxSubjectLength = 50
	maxMessageLength = 5000
)

var validSubjects = map[string]bool{
	"general":  true,
	"feedback": true,
	"bug":      true,
	"other":    true,
}
