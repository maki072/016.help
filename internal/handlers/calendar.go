package handlers

import (
	"fmt"
	"helpdesk/internal/calendar"
	"net/http"
)

func GoogleCalendarAuthHandler(w http.ResponseWriter, r *http.Request) {
	orgID := getOrganizationID(r)
	state := fmt.Sprintf("%d", orgID)
	
	authURL := calendar.GetAuthURL(state)
	if authURL == "" {
		http.Error(w, "Google Calendar not configured", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authURL, http.StatusSeeOther)
}

func GoogleCalendarCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code not provided", http.StatusBadRequest)
		return
	}

	orgID := getOrganizationID(r)

	token, err := calendar.ExchangeCode(code)
	if err != nil {
		http.Error(w, "Failed to exchange code: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := calendar.SaveToken(orgID, token); err != nil {
		http.Error(w, "Failed to save token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
