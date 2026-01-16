package db

import (
	"database/sql"
	"helpdesk/internal/models"
	"time"
)

func SaveGoogleCalendarToken(token *models.GoogleCalendarToken) error {
	query := `
		INSERT INTO google_calendar_tokens (organization_id, access_token, refresh_token, 
		                                   token_type, expiry)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (organization_id) DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_type = EXCLUDED.token_type,
			expiry = EXCLUDED.expiry,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at`
	
	err := DB.QueryRow(query,
		token.OrganizationID, token.AccessToken, token.RefreshToken,
		token.TokenType, token.Expiry,
	).Scan(&token.ID, &token.CreatedAt, &token.UpdatedAt)
	
	return err
}

func GetGoogleCalendarToken(orgID int) (*models.GoogleCalendarToken, error) {
	query := `
		SELECT id, organization_id, access_token, refresh_token, token_type, expiry, 
		       created_at, updated_at
		FROM google_calendar_tokens WHERE organization_id = $1`
	
	token := &models.GoogleCalendarToken{}
	var refreshToken, tokenType sql.NullString
	var expiry sql.NullTime
	
	err := DB.QueryRow(query, orgID).Scan(
		&token.ID, &token.OrganizationID, &token.AccessToken, &refreshToken,
		&tokenType, &expiry, &token.CreatedAt, &token.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if refreshToken.Valid {
		token.RefreshToken = &refreshToken.String
	}
	if tokenType.Valid {
		token.TokenType = &tokenType.String
	}
	if expiry.Valid {
		token.Expiry = &expiry.Time
	}

	return token, nil
}

func UpdateGoogleCalendarToken(orgID int, accessToken, refreshToken string, expiry *time.Time) error {
	query := `
		UPDATE google_calendar_tokens 
		SET access_token = $1, refresh_token = $2, expiry = $3, updated_at = $4
		WHERE organization_id = $5`
	
	_, err := DB.Exec(query, accessToken, refreshToken, expiry, time.Now(), orgID)
	return err
}
