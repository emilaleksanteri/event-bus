package listener

import (
	"time"

	"github.com/google/uuid"
)

type EventBusMessage struct {
	Topic    string    `json:"topic"`
	SenderId string    `json:"client_id"`
	Body     string    `json:"body"`
	SentAt   time.Time `json:"sent_at"`
	Id       uuid.UUID
}
