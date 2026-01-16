package db

import (
	"helpdesk/internal/models"
)

func CreateAttachment(attachment *models.Attachment) error {
	query := `
		INSERT INTO attachments (message_id, file_name, file_path, file_size, mime_type)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	
	err := DB.QueryRow(query,
		attachment.MessageID, attachment.FileName, attachment.FilePath,
		attachment.FileSize, attachment.MimeType,
	).Scan(&attachment.ID, &attachment.CreatedAt)
	
	return err
}

func GetAttachmentsByMessage(messageID int) ([]*models.Attachment, error) {
	query := `
		SELECT id, message_id, file_name, file_path, file_size, mime_type, created_at
		FROM attachments WHERE message_id = $1
		ORDER BY created_at ASC`
	
	rows, err := DB.Query(query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []*models.Attachment
	for rows.Next() {
		attachment := &models.Attachment{}
		var fileSize interface{}
		var mimeType interface{}
		
		err := rows.Scan(
			&attachment.ID, &attachment.MessageID, &attachment.FileName,
			&attachment.FilePath, &fileSize, &mimeType, &attachment.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if fileSize != nil {
			if fs, ok := fileSize.(int64); ok {
				attachment.FileSize = &fs
			}
		}
		if mimeType != nil {
			if mt, ok := mimeType.(string); ok {
				attachment.MimeType = &mt
			}
		}

		attachments = append(attachments, attachment)
	}

	return attachments, rows.Err()
}
