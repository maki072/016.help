package calendar

import (
	"context"
	"fmt"
	"helpdesk/internal/db"
	"helpdesk/internal/models"
	"log"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var oauthConfig *oauth2.Config

func Init() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURI := os.Getenv("GOOGLE_REDIRECT_URI")

	if clientID == "" || clientSecret == "" {
		log.Println("Google Calendar credentials not configured")
		return
	}

	oauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       []string{calendar.CalendarScope},
		Endpoint:     google.Endpoint,
	}
}

func GetAuthURL(state string) string {
	if oauthConfig == nil {
		return ""
	}
	return oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

func ExchangeCode(code string) (*oauth2.Token, error) {
	if oauthConfig == nil {
		return nil, fmt.Errorf("OAuth config not initialized")
	}
	return oauthConfig.Exchange(context.Background(), code)
}

func GetCalendarService(orgID int) (*calendar.Service, error) {
	token, err := db.GetGoogleCalendarToken(orgID)
	if err != nil || token == nil {
		return nil, fmt.Errorf("no calendar token found for organization")
	}

	oauthToken := &oauth2.Token{
		AccessToken: token.AccessToken,
		TokenType:   "Bearer",
	}

	if token.RefreshToken != nil {
		oauthToken.RefreshToken = *token.RefreshToken
	}

	if token.Expiry != nil {
		oauthToken.Expiry = *token.Expiry
	}

	// Check if token needs refresh
	if oauthToken.Expiry.Before(time.Now()) {
		if err := refreshToken(orgID, oauthToken); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
		// Reload token
		token, err = db.GetGoogleCalendarToken(orgID)
		if err != nil || token == nil {
			return nil, fmt.Errorf("failed to reload token")
		}
		oauthToken.AccessToken = token.AccessToken
		if token.Expiry != nil {
			oauthToken.Expiry = *token.Expiry
		}
	}

	ctx := context.Background()
	client := oauthConfig.Client(ctx, oauthToken)
	
	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	return service, nil
}

func refreshToken(orgID int, token *oauth2.Token) error {
	if oauthConfig == nil {
		return fmt.Errorf("OAuth config not initialized")
	}

	ctx := context.Background()
	newToken, err := oauthConfig.TokenSource(ctx, token).Token()
	if err != nil {
		return err
	}

	expiry := newToken.Expiry
	return db.UpdateGoogleCalendarToken(orgID, newToken.AccessToken, newToken.RefreshToken, &expiry)
}

func CreateEvent(orgID int, title, description string, startTime, endTime time.Time) (*calendar.Event, error) {
	service, err := GetCalendarService(orgID)
	if err != nil {
		return nil, err
	}

	org, err := db.GetOrganizationByID(orgID)
	if err != nil {
		return nil, err
	}

	calendarID := "primary"
	if org.GoogleCalendarID != nil && *org.GoogleCalendarID != "" {
		calendarID = *org.GoogleCalendarID
	}

	event := &calendar.Event{
		Summary:     title,
		Description: description,
		Start: &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
			TimeZone: "UTC",
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
			TimeZone: "UTC",
		},
	}

	createdEvent, err := service.Events.Insert(calendarID, event).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return createdEvent, nil
}

func SaveToken(orgID int, token *oauth2.Token) error {
	calendarToken := &models.GoogleCalendarToken{
		OrganizationID: orgID,
		AccessToken:    token.AccessToken,
		TokenType:      &token.TokenType,
		Expiry:         &token.Expiry,
	}

	if token.RefreshToken != "" {
		calendarToken.RefreshToken = &token.RefreshToken
	}

	return db.SaveGoogleCalendarToken(calendarToken)
}
