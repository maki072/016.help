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
var adminIDs map[int64]bool
var operatorIDs map[int64]bool

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

	adminIDs = parseIDList(os.Getenv("TELEGRAM_ADMIN_IDS"))
	operatorIDs = parseIDList(os.Getenv("TELEGRAM_OPERATOR_IDS"))

	log.Printf("Authorized on account %s", BotAPI.Self.UserName)
	return nil
}

func parseIDList(s string) map[int64]bool {
	result := make(map[int64]bool)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if id, err := strconv.ParseInt(part, 10, 64); err == nil && id != 0 {
			result[id] = true
		}
	}
	return result
}

func isAdmin(id int64) bool    { return adminIDs[id] }
func isOperator(id int64) bool { return operatorIDs[id] || adminIDs[id] }

func roleForTelegramID(id int64) string {
	if isAdmin(id) {
		return "admin"
	}
	if operatorIDs[id] {
		return "agent"
	}
	return "customer"
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

	user, err := db.GetUserByTelegramID(chatID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if user == nil {
		org, err := db.GetOrganizationByID(1)
		if err != nil {
			log.Printf("Error getting organization: %v", err)
			return
		}

		username := message.From.UserName
		fullName := strings.TrimSpace(fmt.Sprintf("%s %s", message.From.FirstName, message.From.LastName))
		role := roleForTelegramID(chatID)

		user = &models.User{
			OrganizationID: org.ID,
			TelegramID:     &chatID,
			Username:       &username,
			Role:           role,
			FullName:       &fullName,
			IsActive:       true,
		}

		if err := db.CreateUser(user); err != nil {
			log.Printf("Error creating user: %v", err)
			sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
			return
		}
	}

	if strings.HasPrefix(message.Text, "/") {
		if isOperator(chatID) {
			handleOperatorCommand(message, user)
		} else {
			handleCustomerCommand(message, user)
		}
		return
	}

	// Operators don't create tickets by plain text
	if isOperator(chatID) {
		sendMessage(chatID, "Используйте команды для работы с тикетами. /help — список команд.")
		return
	}

	if message.ReplyToMessage != nil {
		handleReplyToTicket(message, user)
		return
	}

	createTicketFromMessage(message, user)
}

// ─── Customer commands ────────────────────────────────────────────────────────

func handleCustomerCommand(message *tgbotapi.Message, user *models.User) {
	chatID := message.Chat.ID
	text := message.Text

	switch {
	case text == "/start":
		sendMessage(chatID, "Добро пожаловать! Отправьте сообщение, чтобы создать обращение.")
	case text == "/help":
		sendMessage(chatID, "Отправьте сообщение — будет создано обращение.\nОтветьте на сообщение бота, чтобы добавить комментарий.")
	case strings.HasPrefix(text, "/status"):
		handleStatusCommand(message, user)
	default:
		sendMessage(chatID, "Неизвестная команда. Используйте /help.")
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
	if err != nil || ticket == nil {
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
		"resolved":    "Решён",
		"closed":      "Закрыт",
	}
	status := statusText[ticket.Status]
	if status == "" {
		status = ticket.Status
	}

	sendMessage(chatID, fmt.Sprintf("Тикет #%d\nСтатус: %s\nПриоритет: %s", ticket.ID, status, ticket.Priority))
}

// ─── Operator / Admin commands ────────────────────────────────────────────────

func handleOperatorCommand(message *tgbotapi.Message, user *models.User) {
	chatID := message.Chat.ID
	text := message.Text
	parts := strings.Fields(text)
	cmd := parts[0]

	switch cmd {
	case "/start":
		role := "Оператор"
		if isAdmin(chatID) {
			role = "Администратор"
		}
		sendMessage(chatID, fmt.Sprintf("Вы вошли как %s.\n\n/help — список команд.", role))

	case "/help":
		sendMessage(chatID, operatorHelp())

	case "/tickets":
		handleListTickets(chatID, parts)

	case "/mytickets":
		handleMyTickets(chatID, user)

	case "/ticket":
		if len(parts) < 2 {
			sendMessage(chatID, "Использование: /ticket <id>")
			return
		}
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			sendMessage(chatID, "Неверный ID тикета.")
			return
		}
		handleViewTicket(chatID, id)

	case "/reply":
		if len(parts) < 3 {
			sendMessage(chatID, "Использование: /reply <id> <текст ответа>")
			return
		}
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			sendMessage(chatID, "Неверный ID тикета.")
			return
		}
		replyText := strings.Join(parts[2:], " ")
		handleReplyToCustomer(chatID, id, user, replyText)

	case "/assign":
		if len(parts) < 2 {
			sendMessage(chatID, "Использование: /assign <id>")
			return
		}
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			sendMessage(chatID, "Неверный ID тикета.")
			return
		}
		handleAssign(chatID, id, user)

	case "/resolve":
		if len(parts) < 2 {
			sendMessage(chatID, "Использование: /resolve <id>")
			return
		}
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			sendMessage(chatID, "Неверный ID тикета.")
			return
		}
		handleSetStatus(chatID, id, "resolved", "Решён")

	case "/close":
		if len(parts) < 2 {
			sendMessage(chatID, "Использование: /close <id>")
			return
		}
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			sendMessage(chatID, "Неверный ID тикета.")
			return
		}
		handleSetStatus(chatID, id, "closed", "Закрыт")

	case "/reopen":
		if len(parts) < 2 {
			sendMessage(chatID, "Использование: /reopen <id>")
			return
		}
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			sendMessage(chatID, "Неверный ID тикета.")
			return
		}
		handleSetStatus(chatID, id, "open", "Открыт")

	default:
		sendMessage(chatID, "Неизвестная команда. /help — список команд.")
	}
}

func operatorHelp() string {
	return `Команды оператора:

/tickets [open|in_progress|resolved|all] — список тикетов
/mytickets — мои тикеты
/ticket <id> — просмотр тикета
/reply <id> <текст> — ответить клиенту
/assign <id> — взять тикет себе
/resolve <id> — пометить как решённый
/close <id> — закрыть тикет
/reopen <id> — переоткрыть тикет`
}

func handleListTickets(chatID int64, parts []string) {
	statusFilter := "open"
	if len(parts) >= 2 {
		statusFilter = parts[1]
	}

	tickets, err := db.GetTicketsByOrganization(1, statusFilter)
	if err != nil {
		sendMessage(chatID, "Ошибка при получении тикетов.")
		return
	}

	if len(tickets) == 0 {
		sendMessage(chatID, "Тикетов нет.")
		return
	}

	// Show up to 10 tickets
	if len(tickets) > 10 {
		tickets = tickets[:10]
	}

	statusLabel := map[string]string{
		"open":        "Открыт",
		"in_progress": "В работе",
		"resolved":    "Решён",
		"closed":      "Закрыт",
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Тикеты (%s):\n\n", statusFilter))
	for _, t := range tickets {
		st := statusLabel[t.Status]
		if st == "" {
			st = t.Status
		}
		sb.WriteString(fmt.Sprintf("#%d [%s] %s\n", t.ID, st, truncate(t.Title, 50)))
	}
	sb.WriteString("\n/ticket <id> — подробнее")

	sendMessage(chatID, sb.String())
}

func handleMyTickets(chatID int64, user *models.User) {
	tickets, err := db.GetTicketsByAgent(user.ID)
	if err != nil {
		sendMessage(chatID, "Ошибка при получении тикетов.")
		return
	}

	if len(tickets) == 0 {
		sendMessage(chatID, "У вас нет назначенных тикетов.")
		return
	}

	statusLabel := map[string]string{
		"open":        "Открыт",
		"in_progress": "В работе",
		"resolved":    "Решён",
		"closed":      "Закрыт",
	}

	var sb strings.Builder
	sb.WriteString("Мои тикеты:\n\n")
	for _, t := range tickets {
		st := statusLabel[t.Status]
		if st == "" {
			st = t.Status
		}
		sb.WriteString(fmt.Sprintf("#%d [%s] %s\n", t.ID, st, truncate(t.Title, 50)))
	}

	sendMessage(chatID, sb.String())
}

func handleViewTicket(chatID int64, ticketID int) {
	ticket, err := db.GetTicketByID(ticketID)
	if err != nil || ticket == nil {
		sendMessage(chatID, "Тикет не найден.")
		return
	}

	messages, err := db.GetMessagesByTicket(ticketID)
	if err != nil {
		sendMessage(chatID, "Ошибка при получении сообщений.")
		return
	}

	statusLabel := map[string]string{
		"open":        "Открыт",
		"in_progress": "В работе",
		"resolved":    "Решён",
		"closed":      "Закрыт",
	}
	st := statusLabel[ticket.Status]
	if st == "" {
		st = ticket.Status
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Тикет #%d\nСтатус: %s | Приоритет: %s\nТема: %s\n\n", ticket.ID, st, ticket.Priority, ticket.Title))

	// Last 5 messages
	start := 0
	if len(messages) > 5 {
		start = len(messages) - 5
		sb.WriteString(fmt.Sprintf("(показаны последние %d из %d сообщений)\n\n", 5, len(messages)))
	}
	for _, m := range messages[start:] {
		from := "Клиент"
		if !m.IsFromCustomer {
			from = "Оператор"
		}
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", m.CreatedAt.Format("02.01 15:04"), from, truncate(m.Content, 100)))
	}

	sb.WriteString(fmt.Sprintf("\n/reply %d <текст> — ответить\n/assign %d — взять себе\n/resolve %d — решить", ticket.ID, ticket.ID, ticket.ID))

	sendMessage(chatID, sb.String())
}

func handleReplyToCustomer(chatID int64, ticketID int, agent *models.User, text string) {
	ticket, err := db.GetTicketByID(ticketID)
	if err != nil || ticket == nil {
		sendMessage(chatID, "Тикет не найден.")
		return
	}

	msg := &models.Message{
		TicketID:       ticketID,
		UserID:         &agent.ID,
		Content:        text,
		IsFromCustomer: false,
	}

	if err := db.CreateMessage(msg); err != nil {
		log.Printf("Error creating message: %v", err)
		sendMessage(chatID, "Ошибка при отправке ответа.")
		return
	}

	// Update ticket status if open
	if ticket.Status == "open" {
		db.UpdateTicketStatus(ticketID, "in_progress")
	}

	// Send to customer's Telegram if available
	if ticket.TelegramChatID != nil {
		customerMsg := fmt.Sprintf("Ответ по тикету #%d:\n\n%s", ticketID, text)
		sendMessage(*ticket.TelegramChatID, customerMsg)
	}

	sendMessage(chatID, fmt.Sprintf("Ответ отправлен в тикет #%d.", ticketID))
}

func handleAssign(chatID int64, ticketID int, user *models.User) {
	ticket, err := db.GetTicketByID(ticketID)
	if err != nil || ticket == nil {
		sendMessage(chatID, "Тикет не найден.")
		return
	}

	if err := db.AssignTicket(ticketID, user.ID); err != nil {
		log.Printf("Error assigning ticket: %v", err)
		sendMessage(chatID, "Ошибка при назначении тикета.")
		return
	}

	sendMessage(chatID, fmt.Sprintf("Тикет #%d назначен вам.", ticketID))

	// Notify customer
	if ticket.TelegramChatID != nil {
		sendMessage(*ticket.TelegramChatID, fmt.Sprintf("Ваше обращение #%d взято в работу.", ticketID))
	}
}

func handleSetStatus(chatID int64, ticketID int, status, label string) {
	ticket, err := db.GetTicketByID(ticketID)
	if err != nil || ticket == nil {
		sendMessage(chatID, "Тикет не найден.")
		return
	}

	if err := db.UpdateTicketStatus(ticketID, status); err != nil {
		log.Printf("Error updating ticket status: %v", err)
		sendMessage(chatID, "Ошибка при обновлении статуса.")
		return
	}

	sendMessage(chatID, fmt.Sprintf("Тикет #%d: статус изменён на «%s».", ticketID, label))

	// Notify customer
	if ticket.TelegramChatID != nil {
		statusMsg := map[string]string{
			"resolved": fmt.Sprintf("Ваше обращение #%d отмечено как решённое. Если проблема осталась — напишите нам.", ticketID),
			"closed":   fmt.Sprintf("Ваше обращение #%d закрыто.", ticketID),
			"open":     fmt.Sprintf("Ваше обращение #%d переоткрыто.", ticketID),
		}
		if text, ok := statusMsg[status]; ok {
			sendMessage(*ticket.TelegramChatID, text)
		}
	}
}

// ─── Customer ticket creation ─────────────────────────────────────────────────

func createTicketFromMessage(message *tgbotapi.Message, user *models.User) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "" {
		sendMessage(chatID, "Пожалуйста, отправьте текстовое сообщение.")
		return
	}

	ticket := &models.Ticket{
		OrganizationID: user.OrganizationID,
		CustomerID:     &user.ID,
		Title:          truncate(text, 100),
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
		sendMessage(chatID, "Произошла ошибка при создании обращения.")
		return
	}

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
	db.CreateMessage(msg)

	sentMsg := sendMessage(chatID, fmt.Sprintf("Обращение #%d создано. Мы ответим вам в ближайшее время.", ticket.ID))
	if sentMsg != nil && sentMsg.MessageID != 0 {
		msgID := int(sentMsg.MessageID)
		ticket.TelegramMessageID = &msgID
	}

	// Notify all operators
	NotifyOperators(fmt.Sprintf("Новое обращение #%d от %s:\n\n%s", ticket.ID, userName(message.From), truncate(text, 200)))

	log.Printf("New ticket #%d created by user %d", ticket.ID, user.ID)
}

func handleReplyToTicket(message *tgbotapi.Message, user *models.User) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "" {
		return
	}

	repliedMsgID := message.ReplyToMessage.MessageID

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
		sendMessage(chatID, "Не удалось найти тикет для этого сообщения.")
		return
	}

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

	sendMessage(chatID, fmt.Sprintf("Сообщение добавлено к обращению #%d.", ticket.ID))
}

// ─── Callbacks ────────────────────────────────────────────────────────────────

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
	if !isOperator(chatID) {
		return
	}

	user, err := db.GetUserByTelegramID(chatID)
	if err != nil || user == nil {
		return
	}

	switch action {
	case "assign":
		handleAssign(chatID, ticketID, user)
	case "resolve":
		handleSetStatus(chatID, ticketID, "resolved", "Решён")
	}
}

// ─── Public helpers ───────────────────────────────────────────────────────────

// NotifyOperators sends a message to all configured admin and operator Telegram IDs.
func NotifyOperators(text string) {
	if BotAPI == nil {
		return
	}
	notified := make(map[int64]bool)
	for id := range adminIDs {
		if !notified[id] {
			sendMessage(id, text)
			notified[id] = true
		}
	}
	for id := range operatorIDs {
		if !notified[id] {
			sendMessage(id, text)
			notified[id] = true
		}
	}
}

func SendTicketNotification(chatID int64, ticket *models.Ticket, message string) {
	text := fmt.Sprintf("Новое сообщение в обращении #%d:\n\n%s", ticket.ID, message)
	sendMessage(chatID, text)
}

func sendMessage(chatID int64, text string) *tgbotapi.Message {
	msg := tgbotapi.NewMessage(chatID, text)
	sentMsg, err := BotAPI.Send(msg)
	if err != nil {
		log.Printf("Error sending message to %d: %v", chatID, err)
		return nil
	}
	return &sentMsg
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

func userName(from *tgbotapi.User) string {
	if from == nil {
		return "неизвестный"
	}
	name := strings.TrimSpace(fmt.Sprintf("%s %s", from.FirstName, from.LastName))
	if from.UserName != "" {
		name += " (@" + from.UserName + ")"
	}
	return name
}
