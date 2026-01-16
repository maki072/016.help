package handlers

import (
	"encoding/base64"
	"fmt"
	"helpdesk/internal/auth"
	"helpdesk/internal/db"
	"net/http"
	"time"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderTemplate(w, "login.html", nil)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		renderTemplate(w, "login.html", map[string]interface{}{
			"Error": "Email и пароль обязательны",
		})
		return
	}

	user, err := db.GetUserByEmail(email)
	if err != nil || user == nil {
		renderTemplate(w, "login.html", map[string]interface{}{
			"Error": "Неверный email или пароль",
		})
		return
	}

	if user.PasswordHash == nil || !auth.CheckPassword(password, *user.PasswordHash) {
		renderTemplate(w, "login.html", map[string]interface{}{
			"Error": "Неверный email или пароль",
		})
		return
	}

	if !user.IsActive {
		renderTemplate(w, "login.html", map[string]interface{}{
			"Error": "Аккаунт деактивирован",
		})
		return
	}

	sessionID, err := auth.CreateSession(user.ID, user.OrganizationID, user.Role)
	if err != nil {
		http.Error(w, "Ошибка создания сессии", http.StatusInternalServerError)
		return
	}

	cookie := http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		auth.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		Expires:  time.Now(),
		HttpOnly: true,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func RequireAuth(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	session, err := auth.GetSession(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Store session in context (simplified - in production use context)
	r.Header.Set("X-User-ID", fmt.Sprintf("%d", session.UserID))
	r.Header.Set("X-Organization-ID", fmt.Sprintf("%d", session.OrganizationID))
	r.Header.Set("X-User-Role", session.Role)
}

// requireRole is a helper that can be used in future for role-based access control
// Currently not used, but kept for potential future use
func requireRole(role string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			userRole := r.Header.Get("X-User-Role")
			if userRole != role && userRole != "admin" {
				http.Error(w, "Доступ запрещен", http.StatusForbidden)
				return
			}
			next(w, r)
		}
	}
}

func getUserID(r *http.Request) int {
	userIDStr := r.Header.Get("X-User-ID")
	var userID int
	fmt.Sscanf(userIDStr, "%d", &userID)
	return userID
}

func getOrganizationID(r *http.Request) int {
	orgIDStr := r.Header.Get("X-Organization-ID")
	var orgID int
	fmt.Sscanf(orgIDStr, "%d", &orgID)
	return orgID
}

func getUserRole(r *http.Request) string {
	return r.Header.Get("X-User-Role")
}
