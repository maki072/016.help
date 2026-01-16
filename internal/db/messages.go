package db

import (
	"database/sql"
	"helpdesk/internal/models"
)

func CreateMessage(message *models.Message) error {
	query := `
		INSERT INTO messages (ticket_id, user_id, content, telegram_message_id, is_from_customer)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	
	err := DB.QueryRow(query,
		message.TicketID, message.UserID, message.Content,
		message.TelegramMessageID, message.IsFromCustomer,
	).Scan(&message.ID, &message.CreatedAt)
	
	return err
}

func GetMessagesByTicket(ticketID int) ([]*models.Message, error) {
	query := `
		SELECT id, ticket_id, user_id, content, telegram_message_id, is_from_customer, created_at
		FROM messages WHERE ticket_id = $1
		ORDER BY created_at ASC`
	
	rows, err := DB.Query(query, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		message := &models.Message{}
		var userID, telegramMessageID sql.NullInt64
		
		err := rows.Scan(
			&message.ID, &message.TicketID, &userID, &message.Content,
			&telegramMessageID, &message.IsFromCustomer, &message.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if userID.Valid {
			uid := int(userID.Int64)
			message.UserID = &uid
		}
		if telegramMessageID.Valid {
			tmid := int(telegramMessageID.Int64)
			message.TelegramMessageID = &tmid
		}

		messages = append(messages, message)
	}

	return messages, rows.Err()
}
