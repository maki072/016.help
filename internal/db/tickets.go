package db

import (
	"database/sql"
	"helpdesk/internal/models"
	"time"
)

func CreateTicket(ticket *models.Ticket) error {
	query := `
		INSERT INTO tickets (organization_id, customer_id, assigned_agent_id, title, 
		                    description, status, priority, telegram_message_id, telegram_chat_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	err := DB.QueryRow(query,
		ticket.OrganizationID, ticket.CustomerID, ticket.AssignedAgentID,
		ticket.Title, ticket.Description, ticket.Status, ticket.Priority,
		ticket.TelegramMessageID, ticket.TelegramChatID,
	).Scan(&ticket.ID, &ticket.CreatedAt, &ticket.UpdatedAt)

	return err
}

func GetTicketByID(id int) (*models.Ticket, error) {
	query := `
		SELECT id, organization_id, customer_id, assigned_agent_id, title, description,
		       status, priority, telegram_message_id, telegram_chat_id, created_at, updated_at
		FROM tickets WHERE id = $1`

	ticket := &models.Ticket{}
	var customerID, assignedAgentID, telegramMessageID sql.NullInt64
	var description sql.NullString
	var telegramChatID sql.NullInt64

	err := DB.QueryRow(query, id).Scan(
		&ticket.ID, &ticket.OrganizationID, &customerID, &assignedAgentID,
		&ticket.Title, &description, &ticket.Status, &ticket.Priority,
		&telegramMessageID, &telegramChatID, &ticket.CreatedAt, &ticket.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if customerID.Valid {
		cid := int(customerID.Int64)
		ticket.CustomerID = &cid
	}
	if assignedAgentID.Valid {
		aid := int(assignedAgentID.Int64)
		ticket.AssignedAgentID = &aid
	}
	if description.Valid {
		ticket.Description = &description.String
	}
	if telegramMessageID.Valid {
		tmid := int(telegramMessageID.Int64)
		ticket.TelegramMessageID = &tmid
	}
	if telegramChatID.Valid {
		ticket.TelegramChatID = &telegramChatID.Int64
	}

	return ticket, nil
}

func GetTicketByTelegramMessage(chatID int64, messageID int) (*models.Ticket, error) {
	query := `
		SELECT id, organization_id, customer_id, assigned_agent_id, title, description,
		       status, priority, telegram_message_id, telegram_chat_id, created_at, updated_at
		FROM tickets WHERE telegram_chat_id = $1 AND telegram_message_id = $2`

	ticket := &models.Ticket{}
	var customerID, assignedAgentID, telegramMessageID sql.NullInt64
	var description sql.NullString
	var telegramChatID sql.NullInt64

	err := DB.QueryRow(query, chatID, messageID).Scan(
		&ticket.ID, &ticket.OrganizationID, &customerID, &assignedAgentID,
		&ticket.Title, &description, &ticket.Status, &ticket.Priority,
		&telegramMessageID, &telegramChatID, &ticket.CreatedAt, &ticket.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if customerID.Valid {
		cid := int(customerID.Int64)
		ticket.CustomerID = &cid
	}
	if assignedAgentID.Valid {
		aid := int(assignedAgentID.Int64)
		ticket.AssignedAgentID = &aid
	}
	if description.Valid {
		ticket.Description = &description.String
	}
	if telegramMessageID.Valid {
		tmid := int(telegramMessageID.Int64)
		ticket.TelegramMessageID = &tmid
	}
	if telegramChatID.Valid {
		ticket.TelegramChatID = &telegramChatID.Int64
	}

	return ticket, nil
}

func GetTicketsByOrganization(orgID int, statusFilter string) ([]*models.Ticket, error) {
	var query string
	var args []interface{}

	if statusFilter != "" && statusFilter != "all" {
		query = `
			SELECT id, organization_id, customer_id, assigned_agent_id, title, description,
			       status, priority, telegram_message_id, telegram_chat_id, created_at, updated_at
			FROM tickets WHERE organization_id = $1 AND status = $2
			ORDER BY created_at DESC`
		args = []interface{}{orgID, statusFilter}
	} else {
		query = `
			SELECT id, organization_id, customer_id, assigned_agent_id, title, description,
			       status, priority, telegram_message_id, telegram_chat_id, created_at, updated_at
			FROM tickets WHERE organization_id = $1
			ORDER BY created_at DESC`
		args = []interface{}{orgID}
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*models.Ticket
	for rows.Next() {
		ticket := &models.Ticket{}
		var customerID, assignedAgentID, telegramMessageID sql.NullInt64
		var description sql.NullString
		var telegramChatID sql.NullInt64

		err := rows.Scan(
			&ticket.ID, &ticket.OrganizationID, &customerID, &assignedAgentID,
			&ticket.Title, &description, &ticket.Status, &ticket.Priority,
			&telegramMessageID, &telegramChatID, &ticket.CreatedAt, &ticket.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if customerID.Valid {
			cid := int(customerID.Int64)
			ticket.CustomerID = &cid
		}
		if assignedAgentID.Valid {
			aid := int(assignedAgentID.Int64)
			ticket.AssignedAgentID = &aid
		}
		if description.Valid {
			ticket.Description = &description.String
		}
		if telegramMessageID.Valid {
			tmid := int(telegramMessageID.Int64)
			ticket.TelegramMessageID = &tmid
		}
		if telegramChatID.Valid {
			ticket.TelegramChatID = &telegramChatID.Int64
		}

		tickets = append(tickets, ticket)
	}

	return tickets, rows.Err()
}

func UpdateTicketStatus(id int, status string) error {
	query := `UPDATE tickets SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := DB.Exec(query, status, time.Now(), id)
	return err
}

func AssignTicket(ticketID, agentID int) error {
	query := `UPDATE tickets SET assigned_agent_id = $1, status = 'in_progress', updated_at = $2 WHERE id = $3`
	_, err := DB.Exec(query, agentID, time.Now(), ticketID)
	return err
}

func GetTicketsByAgent(agentID int) ([]*models.Ticket, error) {
	query := `
		SELECT id, organization_id, customer_id, assigned_agent_id, title, description,
		       status, priority, telegram_message_id, telegram_chat_id, created_at, updated_at
		FROM tickets WHERE assigned_agent_id = $1
		ORDER BY created_at DESC`

	rows, err := DB.Query(query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*models.Ticket
	for rows.Next() {
		ticket := &models.Ticket{}
		var customerID, assignedAgentID, telegramMessageID sql.NullInt64
		var description sql.NullString
		var telegramChatID sql.NullInt64

		err := rows.Scan(
			&ticket.ID, &ticket.OrganizationID, &customerID, &assignedAgentID,
			&ticket.Title, &description, &ticket.Status, &ticket.Priority,
			&telegramMessageID, &telegramChatID, &ticket.CreatedAt, &ticket.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if customerID.Valid {
			cid := int(customerID.Int64)
			ticket.CustomerID = &cid
		}
		if assignedAgentID.Valid {
			aid := int(assignedAgentID.Int64)
			ticket.AssignedAgentID = &aid
		}
		if description.Valid {
			ticket.Description = &description.String
		}
		if telegramMessageID.Valid {
			tmid := int(telegramMessageID.Int64)
			ticket.TelegramMessageID = &tmid
		}
		if telegramChatID.Valid {
			ticket.TelegramChatID = &telegramChatID.Int64
		}

		tickets = append(tickets, ticket)
	}

	return tickets, rows.Err()
}
