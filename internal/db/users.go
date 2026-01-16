package db

import (
	"database/sql"
	"helpdesk/internal/models"
	"time"
)

func GetUserByID(id int) (*models.User, error) {
	query := `
		SELECT id, organization_id, telegram_id, username, email, password_hash, 
		       role, full_name, is_active, created_at, updated_at
		FROM users WHERE id = $1`

	user := &models.User{}
	var telegramID sql.NullInt64
	var username, email, passwordHash, fullName sql.NullString

	err := DB.QueryRow(query, id).Scan(
		&user.ID, &user.OrganizationID, &telegramID, &username, &email,
		&passwordHash, &user.Role, &fullName, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if telegramID.Valid {
		user.TelegramID = &telegramID.Int64
	}
	if username.Valid {
		user.Username = &username.String
	}
	if email.Valid {
		user.Email = &email.String
	}
	if passwordHash.Valid {
		user.PasswordHash = &passwordHash.String
	}
	if fullName.Valid {
		user.FullName = &fullName.String
	}

	return user, nil
}

func GetUserByTelegramID(telegramID int64) (*models.User, error) {
	query := `
		SELECT id, organization_id, telegram_id, username, email, password_hash, 
		       role, full_name, is_active, created_at, updated_at
		FROM users WHERE telegram_id = $1`

	user := &models.User{}
	var tgID sql.NullInt64
	var username, email, passwordHash, fullName sql.NullString

	err := DB.QueryRow(query, telegramID).Scan(
		&user.ID, &user.OrganizationID, &tgID, &username, &email,
		&passwordHash, &user.Role, &fullName, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if tgID.Valid {
		user.TelegramID = &tgID.Int64
	}
	if username.Valid {
		user.Username = &username.String
	}
	if email.Valid {
		user.Email = &email.String
	}
	if passwordHash.Valid {
		user.PasswordHash = &passwordHash.String
	}
	if fullName.Valid {
		user.FullName = &fullName.String
	}

	return user, nil
}

func GetUserByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, organization_id, telegram_id, username, email, password_hash, 
		       role, full_name, is_active, created_at, updated_at
		FROM users WHERE email = $1`

	user := &models.User{}
	var telegramID sql.NullInt64
	var username, emailVal, passwordHash, fullName sql.NullString

	err := DB.QueryRow(query, email).Scan(
		&user.ID, &user.OrganizationID, &telegramID, &username, &emailVal,
		&passwordHash, &user.Role, &fullName, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if telegramID.Valid {
		user.TelegramID = &telegramID.Int64
	}
	if username.Valid {
		user.Username = &username.String
	}
	if emailVal.Valid {
		user.Email = &emailVal.String
	}
	if passwordHash.Valid {
		user.PasswordHash = &passwordHash.String
	}
	if fullName.Valid {
		user.FullName = &fullName.String
	}

	return user, nil
}

func CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (organization_id, telegram_id, username, email, password_hash, 
		                  role, full_name, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	err := DB.QueryRow(query,
		user.OrganizationID, user.TelegramID, user.Username, user.Email,
		user.PasswordHash, user.Role, user.FullName, user.IsActive,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	return err
}

func UpdateUserTelegramID(userID int, telegramID int64) error {
	query := `UPDATE users SET telegram_id = $1, updated_at = $2 WHERE id = $3`
	_, err := DB.Exec(query, telegramID, time.Now(), userID)
	return err
}

func GetUsersByOrganization(orgID int) ([]*models.User, error) {
	query := `
		SELECT id, organization_id, telegram_id, username, email, password_hash, 
		       role, full_name, is_active, created_at, updated_at
		FROM users WHERE organization_id = $1 AND is_active = TRUE
		ORDER BY created_at DESC`

	rows, err := DB.Query(query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var telegramID sql.NullInt64
		var username, email, passwordHash, fullName sql.NullString

		err := rows.Scan(
			&user.ID, &user.OrganizationID, &telegramID, &username, &email,
			&passwordHash, &user.Role, &fullName, &user.IsActive,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if telegramID.Valid {
			user.TelegramID = &telegramID.Int64
		}
		if username.Valid {
			user.Username = &username.String
		}
		if email.Valid {
			user.Email = &email.String
		}
		if passwordHash.Valid {
			user.PasswordHash = &passwordHash.String
		}
		if fullName.Valid {
			user.FullName = &fullName.String
		}

		users = append(users, user)
	}

	return users, rows.Err()
}
