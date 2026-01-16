package db

import (
	"database/sql"
	"helpdesk/internal/models"
)

func GetOrganizationByID(id int) (*models.Organization, error) {
	query := `
		SELECT id, name, telegram_chat_id, google_calendar_id, created_at, updated_at
		FROM organizations WHERE id = $1`
	
	org := &models.Organization{}
	var telegramChatID sql.NullInt64
	var googleCalendarID sql.NullString
	
	err := DB.QueryRow(query, id).Scan(
		&org.ID, &org.Name, &telegramChatID, &googleCalendarID,
		&org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if telegramChatID.Valid {
		org.TelegramChatID = &telegramChatID.Int64
	}
	if googleCalendarID.Valid {
		org.GoogleCalendarID = &googleCalendarID.String
	}

	return org, nil
}

func GetAllOrganizations() ([]*models.Organization, error) {
	query := `
		SELECT id, name, telegram_chat_id, google_calendar_id, created_at, updated_at
		FROM organizations ORDER BY created_at DESC`
	
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []*models.Organization
	for rows.Next() {
		org := &models.Organization{}
		var telegramChatID sql.NullInt64
		var googleCalendarID sql.NullString
		
		err := rows.Scan(
			&org.ID, &org.Name, &telegramChatID, &googleCalendarID,
			&org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if telegramChatID.Valid {
			org.TelegramChatID = &telegramChatID.Int64
		}
		if googleCalendarID.Valid {
			org.GoogleCalendarID = &googleCalendarID.String
		}

		orgs = append(orgs, org)
	}

	return orgs, rows.Err()
}
