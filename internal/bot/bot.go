package bot

import (
	"fmt"
	"helpdesk/internal/db"
	"helpdesk/internal/models"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var BotAPI *tgbotapi.BotAPI

func Init() error {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN is not set")
	}

	var err error
	BotAPI, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	log.Printf("Authorized on account %s", BotAPI.Self.UserName)
	return nil
}

func Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := BotAPI.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			handleCallback(update.CallbackQuery)
		}
	}
}

func handleMessage(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	// Check if user exists
	user, err := db.GetUserByTelegramID(chatID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	// If user doesn't exist, create customer
	if user == nil {
		org, err := db.GetOrganizationByID(1) // Default organization
		if err != nil {
			log.Printf("Error getting organization: %v", err)
			return
		}

		username := message.From.UserName
		fullName := fmt.Sprintf("%s %s", message.From.FirstName, message.From.LastName)
		
		user = &models.User{
			OrganizationID: org.ID,
			TelegramID:    &chatID,
			Username:      &username,
			Role:          "customer",
			FullName:      &fullName,
			IsActive:      true,
		}

		if err := db.CreateUser(user); err != nil {
			log.Printf("Error creating user: %v", err)
			sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
			return
		}
	}

	// Handle commands
	if strings.HasPrefix(text, "/") {
		handleCommand(message, user)
		return
	}

	// Handle reply to ticket message
	if message.ReplyToMessage != nil {
		handleReplyToTicket(message, user)
		return
	}

	// Create new ticket
	createTicketFromMessage(message, user)
}

func handleCommand(message *tgbotapi.Message, user *models.User) {
	text := message.Text
	chatID := message.Chat.ID

	switch {
	case text == "/start":
		sendMessage(chatID, "Добро пожаловать в Helpdesk! Отправьте сообщение, чтобы создать тикет.")
	case text == "/help":
		sendMessage(chatID, "Отправьте сообщение, чтобы создать новый тикет.\nОтветьте на сообщение бота, чтобы добавить комментарий к тикету.")
	case strings.HasPrefix(text, "/status"):
		handleStatusCommand(message, user)
	default:
		sendMessage(chatID, "Неизвестная команда. Используйте /help для справки.")
	}
}

func handleStatusCommand(message *tgbotapi.Message, user *models.User) {
	chatID := message.Chat.ID
	parts := strings.Fields(message.Text)
	
	if len(parts) < 2 {
		sendMessage(chatID, "Использование: /status <номер_тикета>")
		return
	}

	ticketID, err := strconv.Atoi(parts[1])
	if err != nil {
		sendMessage(chatID, "Неверный номер тикета.")
		return
	}

	ticket, err := db.GetTicketByID(ticketID)
	if err != nil {
		sendMessage(chatID, "Тикет не найден.")
		return
	}

	if ticket.CustomerID == nil || *ticket.CustomerID != user.ID {
		sendMessage(chatID, "У вас нет доступа к этому тикету.")
		return
	}

	statusText := map[string]string{
		"open":        "Открыт",
		"in_progress": "В работе",
		"resolved":    "Решен",
		"closed":      "Закрыт",
	}

	status := statusText[ticket.Status]
	if status == "" {
		status = ticket.Status
	}

	msg := fmt.Sprintf("Тикет #%d\nСтатус: %s\nПриоритет: %s", ticket.ID, status, ticket.Priority)
	sendMessage(chatID, msg)
}

func createTicketFromMessage(message *tgbotapi.Message, user *models.User) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "" {
		sendMessage(chatID, "Пожалуйста, отправьте текстовое сообщение для создания тикета.")
		return
	}

	// Create ticket
	ticket := &models.Ticket{
		OrganizationID: user.OrganizationID,
		CustomerID:     &user.ID,
		Title:          text,
		Description:    &text,
		Status:         "open",
		Priority:       "medium",
		TelegramChatID: &chatID,
	}

	if message.MessageID != 0 {
		msgID := int(message.MessageID)
		ticket.TelegramMessageID = &msgID
	}

	if err := db.CreateTicket(ticket); err != nil {
		log.Printf("Error creating ticket: %v", err)
		sendMessage(chatID, "Произошла ошибка при создании тикета.")
		return
	}

	// Create initial message
	msg := &models.Message{
		TicketID:       ticket.ID,
		UserID:         &user.ID,
		Content:        text,
		IsFromCustomer: true,
	}

	if message.MessageID != 0 {
		msgID := int(message.MessageID)
		msg.TelegramMessageID = &msgID
	}

	if err := db.CreateMessage(msg); err != nil {
		log.Printf("Error creating message: %v", err)
	}

	// Send confirmation
	reply := fmt.Sprintf("Тикет #%d создан успешно!\n\nВаше сообщение: %s", ticket.ID, text)
	sentMsg := sendMessage(chatID, reply)

	// Update ticket with bot's message ID
	if sentMsg != nil && sentMsg.MessageID != 0 {
		msgID := int(sentMsg.MessageID)
		ticket.TelegramMessageID = &msgID
	}

	// Notify agents (in real implementation, you'd notify assigned agents)
	log.Printf("New ticket #%d created by user %d", ticket.ID, user.ID)
}

func handleReplyToTicket(message *tgbotapi.Message, user *models.User) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "" {
		return
	}

	// Try to find ticket by replied message
	repliedMsgID := message.ReplyToMessage.MessageID
	
	// Search for ticket with this message ID
	tickets, err := db.GetTicketsByOrganization(user.OrganizationID, "")
	if err != nil {
		log.Printf("Error getting tickets: %v", err)
		return
	}

	var ticket *models.Ticket
	for _, t := range tickets {
		if t.TelegramMessageID != nil && *t.TelegramMessageID == repliedMsgID {
			ticket = t
			break
		}
	}

	if ticket == nil {
		sendMessage(chatID, "Тикет не найден.")
		return
	}

	// Create message
	msg := &models.Message{
		TicketID:       ticket.ID,
		UserID:         &user.ID,
		Content:        text,
		IsFromCustomer: user.Role == "customer",
	}

	if message.MessageID != 0 {
		msgID := int(message.MessageID)
		msg.TelegramMessageID = &msgID
	}

	if err := db.CreateMessage(msg); err != nil {
		log.Printf("Error creating message: %v", err)
		sendMessage(chatID, "Ошибка при добавлении сообщения.")
		return
	}

	sendMessage(chatID, fmt.Sprintf("Сообщение добавлено к тикету #%d", ticket.ID))
}

func handleCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	if strings.HasPrefix(data, "ticket_") {
		parts := strings.Split(data, "_")
		if len(parts) >= 3 {
			action := parts[1]
			ticketID, err := strconv.Atoi(parts[2])
			if err == nil {
				handleTicketAction(action, ticketID, chatID, callback)
			}
		}
	}

	BotAPI.Request(tgbotapi.NewCallback(callback.ID, ""))
}

func handleTicketAction(action string, ticketID int, chatID int64, callback *tgbotapi.CallbackQuery) {
	ticket, err := db.GetTicketByID(ticketID)
	if err != nil {
		sendMessage(chatID, "Тикет не найден.")
		return
	}

	switch action {
	case "assign":
		user, err := db.GetUserByTelegramID(chatID)
		if err == nil && user != nil && (user.Role == "admin" || user.Role == "agent") {
			db.AssignTicket(ticketID, user.ID)
			sendMessage(chatID, fmt.Sprintf("Тикет #%d назначен вам.", ticketID))
		}
	case "resolve":
		user, err := db.GetUserByTelegramID(chatID)
		if err == nil && user != nil && (user.Role == "admin" || user.Role == "agent") {
			db.UpdateTicketStatus(ticketID, "resolved")
			sendMessage(chatID, fmt.Sprintf("Тикет #%d помечен как решенный.", ticketID))
		}
	}
}

func sendMessage(chatID int64, text string) *tgbotapi.Message {
	msg := tgbotapi.NewMessage(chatID, text)
	sentMsg, err := BotAPI.Send(msg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return nil
	}
	return &sentMsg
}

func SendTicketNotification(chatID int64, ticket *models.Ticket, message string) {
	text := fmt.Sprintf("Новое сообщение в тикете #%d:\n\n%s", ticket.ID, message)
	sendMessage(chatID, text)
}
