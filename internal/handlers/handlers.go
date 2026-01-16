package handlers

import (
	"fmt"
	"helpdesk/internal/db"
	"helpdesk/internal/models"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

var templates *template.Template

func InitTemplates() error {
	tmplFiles, err := filepath.Glob("templates/*.html")
	if err != nil {
		return err
	}

	templates, err = template.ParseFiles(tmplFiles...)
	return err
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	if templates == nil {
		http.Error(w, "Templates not initialized", http.StatusInternalServerError)
		return
	}

	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	orgID := getOrganizationID(r)
	userRole := getUserRole(r)

	statusFilter := r.URL.Query().Get("status")
	if statusFilter == "" {
		statusFilter = "all"
	}

	tickets, err := db.GetTicketsByOrganization(orgID, statusFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Tickets":     tickets,
		"StatusFilter": statusFilter,
		"UserRole":    userRole,
	}

	renderTemplate(w, "dashboard.html", data)
}

func TicketHandler(w http.ResponseWriter, r *http.Request) {
	ticketIDStr := r.URL.Path[len("/ticket/"):]
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	ticket, err := db.GetTicketByID(ticketID)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	orgID := getOrganizationID(r)
	if ticket.OrganizationID != orgID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	messages, err := db.GetMessagesByTicket(ticketID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get attachments for each message
	for _, msg := range messages {
		attachments, _ := db.GetAttachmentsByMessage(msg.ID)
		// Store in a map for template access
		_ = attachments
	}

	users, err := db.GetUsersByOrganization(orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Ticket":   ticket,
		"Messages": messages,
		"Users":    users,
		"UserRole": getUserRole(r),
	}

	renderTemplate(w, "ticket.html", data)
}

func AddMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ticketIDStr := r.FormValue("ticket_id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	ticket, err := db.GetTicketByID(ticketID)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	orgID := getOrganizationID(r)
	if ticket.OrganizationID != orgID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	userID := getUserID(r)
	userRole := getUserRole(r)

	message := &models.Message{
		TicketID:       ticketID,
		UserID:         &userID,
		Content:        content,
		IsFromCustomer: userRole == "customer",
	}

	if err := db.CreateMessage(message); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update ticket status if needed
	if ticket.Status == "open" && userRole != "customer" {
		db.UpdateTicketStatus(ticketID, "in_progress")
	}

	http.Redirect(w, r, fmt.Sprintf("/ticket/%d", ticketID), http.StatusSeeOther)
}

func UpdateTicketStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ticketIDStr := r.FormValue("ticket_id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	status := r.FormValue("status")
	if status == "" {
		http.Error(w, "Status is required", http.StatusBadRequest)
		return
	}

	ticket, err := db.GetTicketByID(ticketID)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	orgID := getOrganizationID(r)
	if ticket.OrganizationID != orgID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	userRole := getUserRole(r)
	if userRole == "customer" {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if err := db.UpdateTicketStatus(ticketID, status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/ticket/%d", ticketID), http.StatusSeeOther)
}

func AssignTicketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ticketIDStr := r.FormValue("ticket_id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	agentIDStr := r.FormValue("agent_id")
	agentID, err := strconv.Atoi(agentIDStr)
	if err != nil {
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	ticket, err := db.GetTicketByID(ticketID)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	orgID := getOrganizationID(r)
	if ticket.OrganizationID != orgID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if err := db.AssignTicket(ticketID, agentID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/ticket/%d", ticketID), http.StatusSeeOther)
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// File upload handling would go here
	// For simplicity, we'll skip it in this basic implementation
	http.Error(w, "File upload not implemented", http.StatusNotImplemented)
}

func StaticHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/static/"):]
	filePath := filepath.Join("static", path)
	
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, filePath)
}
