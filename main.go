package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

const TOOL_ERROR_AUTHENTICATION_REQUIRED = "plz authenticate with Google Calendar and retry"

var (
	oauthConfig *oauth2.Config
	state = "stateless" // could be a secure random value for production
	calendarService *calendar.Service
)

func main() {
	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost" // for binding
	}
	advertisedHost := os.Getenv("ADVERTISED_HOST")
	if advertisedHost == "" {
		advertisedHost = "localhost" // for client connections
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "5555"
	}

	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  fmt.Sprintf("http://%s:%s/auth/callback", advertisedHost, port),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",       // For SSO
			"https://www.googleapis.com/auth/userinfo.profile",     // For SSO
			"https://www.googleapis.com/auth/calendar.readonly",    // For Calendar
			"openid",                                                // OpenID for ID token
		},
		Endpoint: google.Endpoint,
	}

	// Create a new MCP server
	mcpServer := server.NewMCPServer(
		"Google Calendar MCP", // Name of the server
		"0.1.0",               // Version
		// Set listChanged to false as this example
		// server does not emit notifications
		// when the list of available tools changes
		server.WithToolCapabilities(false),
	)

	// Define tools
	setupTools(mcpServer)

	sseServer := server.NewSSEServer(mcpServer,
		server.WithBaseURL(fmt.Sprintf("http://%s:%s", advertisedHost, port)),
		server.WithStaticBasePath("/mcp"),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", handleAuthCallback(mcpServer))
	mux.Handle("/mcp/sse", sseServer.SSEHandler())
	mux.Handle("/mcp/message", sseServer.MessageHandler())

	log.Printf("Server listening at http://%s:%s", host, port)
	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", host, port), mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
func handleAuthCallback(server *server.MCPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		token, err := oauthConfig.Exchange(context.Background(), code)
		if err != nil {
			fmt.Println("token exchange failed")
			http.Error(w, "token exchange failed", http.StatusInternalServerError)
			return
		}

		forMethod := r.URL.Query().Get("state")
		if state == "" {
			fmt.Println("state is required")
			http.Error(w, "state is required", http.StatusBadRequest)
			return
		}

		// Initialize the global calendarService
		tokenSource := oauthConfig.TokenSource(context.Background(), token)
		srv, err := calendar.NewService(context.Background(), option.WithTokenSource(tokenSource))
		if err != nil {
			http.Error(w, fmt.Sprintf("Unable to create Calendar service: %v", err), http.StatusInternalServerError)
			return
		}
		calendarService = srv

		// Notify the client to continue the method that requested authentication
		// Note: Some MCP clients may not support this yet. e.g. Cursor
		fmt.Println("sending notification to client")
		server.SendNotificationToClient(
			context.Background(),
			forMethod,
			map[string]any{},
		)
		fmt.Println("sent notification to client")
	}
}

func setupTools(s *server.MCPServer) {
	// Lazy auth tool
	authTool := mcp.NewTool("auth",
		mcp.WithDescription("Authenticate with Google Calendar to use the other tools. Use it when you run into authentication issues."),
		mcp.WithString("for_method",
			mcp.Description("The method that requires authentication"),
		),
	)

	s.AddTool(authTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// FIXME: Protocol support of multi-turn tool responses is not finalized / implemented yet.
		// Latest support: https://modelcontextprotocol.io/specification/2025-03-26/basic/utilities/progress
		// Discussion: https://github.com/modelcontextprotocol/modelcontextprotocol/discussions/314
		// Difference of MCP and A2D: https://docs.google.com/document/d/18aW0P38h14ccJTkN-irhy2NVdG-_V_Oe7WblnAiKKGM/edit?tab=t.0

		// session := server.ClientSessionFromContext(ctx)
		// url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
		// session.NotificationChannel() <- mcp.JSONRPCNotification{
		// 	Notification: mcp.Notification{
		// 		Method: "Please visit this URL to authenticate: " + url,
		// 	},
		// }

		// for {
		// 	select {
		// 	case <-ctx.Done():
		// 		if calendarService != nil {
		// 			return mcp.NewToolResultText("Authentication timed out. Please try again."), nil
		// 		}
		// 		break
		// 	default:
		// 		if calendarService != nil {
		// 			break
		// 		}
		// 		time.Sleep(2 * time.Second)
		// 	}
		// }

		args := request.Params.Arguments.(map[string]any)
		forMethod, _ := args["for_method"].(string)

		// Copy oauthConfig and add for_method to the redirect URL
		newConfig := *oauthConfig // shallow copy is fine since all fields are value or immutable
		// FIXME: Google OAuth2 is strict about the redirect URL, so we need to use a different approach
		// redirectURL, _ := url.Parse(oauthConfig.RedirectURL)
		// query := redirectURL.Query()
		// query.Set("for_method", forMethod)
		// redirectURL.RawQuery = query.Encode()
		// fmt.Println(redirectURL.String())
		// newConfig.RedirectURL = redirectURL.String()

		url := newConfig.AuthCodeURL(forMethod, oauth2.AccessTypeOffline)
		return mcp.NewToolResultText(fmt.Sprintf("Please visit this URL to authenticate: %s", url)), nil
	})

	// Get current time tool
	getCurrentTimeTool := mcp.NewTool("get_current_time",
		mcp.WithDescription("Get the current time in a specific timezone"),
		mcp.WithString("timezone",
			mcp.Description("The timezone to get the current time in (e.g., 'America/New_York', 'UTC', 'Asia/Tokyo')"),
			mcp.DefaultString("UTC"),
		),
	)

	s.AddTool(getCurrentTimeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]any)
		timezone := args["timezone"].(string)

		loc, err := time.LoadLocation(timezone)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Invalid timezone: %v", err)), nil
		}

		currentTime := time.Now().In(loc)
		return mcp.NewToolResultText(fmt.Sprintf("Current time in %s: %s", timezone, currentTime.Format(time.RFC3339))), nil
	})

	// List calendars tool
	listCalendarsTool := mcp.NewTool("list_calendars",
		mcp.WithDescription("List all accessible Google Calendars"),
	)

	s.AddTool(listCalendarsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if calendarService == nil {
			return mcp.NewToolResultError(TOOL_ERROR_AUTHENTICATION_REQUIRED), nil
		}
		calendarList, err := calendarService.CalendarList.List().Do()
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error listing calendars: %v", err)), nil
		}

		result := "Available Calendars:\n"
		for _, item := range calendarList.Items {
			result += fmt.Sprintf("- %s (ID: %s)\n", item.Summary, item.Id)
		}

		return mcp.NewToolResultText(result), nil
	})

	// List events tool
	listEventsTool := mcp.NewTool("list_events",
		mcp.WithDescription("List events from a Google Calendar"),
		mcp.WithString("calendar_id",
			mcp.Description("The calendar ID (use 'primary' for primary calendar)"),
			mcp.DefaultString("primary"),
		),
		mcp.WithString("time_min",
			mcp.Description("Lower bound for event start time (RFC3339 format, e.g., '2024-01-01T00:00:00Z')"),
		),
		mcp.WithString("time_max",
			mcp.Description("Upper bound for event start time (RFC3339 format, e.g., '2024-12-31T23:59:59Z')"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of events to return"),
			mcp.DefaultNumber(10),
		),
	)

	s.AddTool(listEventsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if calendarService == nil {
			return mcp.NewToolResultError(TOOL_ERROR_AUTHENTICATION_REQUIRED), nil
		}
		args := request.Params.Arguments.(map[string]any)
		calendarID := args["calendar_id"].(string)
		maxResults := int64(args["max_results"].(float64))

		call := calendarService.Events.List(calendarID).
			MaxResults(maxResults).
			SingleEvents(true).
			OrderBy("startTime")

		if timeMin, ok := args["time_min"].(string); ok && timeMin != "" {
			call = call.TimeMin(timeMin)
		}

		if timeMax, ok := args["time_max"].(string); ok && timeMax != "" {
			call = call.TimeMax(timeMax)
		}

		events, err := call.Do()
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error listing events: %v", err)), nil
		}

		if len(events.Items) == 0 {
			return mcp.NewToolResultText("No events found."), nil
		}

		result := fmt.Sprintf("Events in calendar %s:\n", calendarID)
		for _, item := range events.Items {
			date := item.Start.DateTime
			if date == "" {
				date = item.Start.Date
			}
			result += fmt.Sprintf("- %s (%s)\n", item.Summary, date)
		}

		return mcp.NewToolResultText(result), nil
	})

	// Create event tool
	createEventTool := mcp.NewTool("create_event",
		mcp.WithDescription("Create a new event in Google Calendar"),
		mcp.WithString("calendar_id",
			mcp.Description("The calendar ID (use 'primary' for primary calendar)"),
			mcp.DefaultString("primary"),
		),
		mcp.WithString("summary",
			mcp.Description("Event title/summary"),
		),
		mcp.WithString("description",
			mcp.Description("Event description (optional)"),
		),
		mcp.WithString("start_time",
			mcp.Description("Event start time (RFC3339 format, e.g., '2024-01-01T10:00:00Z')"),
		),
		mcp.WithString("end_time",
			mcp.Description("Event end time (RFC3339 format, e.g., '2024-01-01T11:00:00Z')"),
		),
		mcp.WithString("location",
			mcp.Description("Event location (optional)"),
		),
	)

	s.AddTool(createEventTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if calendarService == nil {
			return mcp.NewToolResultError(TOOL_ERROR_AUTHENTICATION_REQUIRED), nil
		}
		args := request.Params.Arguments.(map[string]any)
		calendarID := args["calendar_id"].(string)
		summary := args["summary"].(string)
		startTime := args["start_time"].(string)
		endTime := args["end_time"].(string)

		event := &calendar.Event{
			Summary: summary,
			Start: &calendar.EventDateTime{
				DateTime: startTime,
			},
			End: &calendar.EventDateTime{
				DateTime: endTime,
			},
		}

		if description, ok := args["description"].(string); ok && description != "" {
			event.Description = description
		}

		if location, ok := args["location"].(string); ok && location != "" {
			event.Location = location
		}

		createdEvent, err := calendarService.Events.Insert(calendarID, event).Do()
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error creating event: %v", err)), nil
		}

		result := fmt.Sprintf("Event created successfully!\nTitle: %s\nID: %s\nHTML Link: %s",
			createdEvent.Summary, createdEvent.Id, createdEvent.HtmlLink)

		return mcp.NewToolResultText(result), nil
	})

	// Get event details tool
	getEventTool := mcp.NewTool("get_event",
		mcp.WithDescription("Get details of a specific event"),
		mcp.WithString("calendar_id",
			mcp.Description("The calendar ID (use 'primary' for primary calendar)"),
			mcp.DefaultString("primary"),
		),
		mcp.WithString("event_id",
			mcp.Description("The event ID"),
		),
	)

	s.AddTool(getEventTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if calendarService == nil {
			return mcp.NewToolResultError(TOOL_ERROR_AUTHENTICATION_REQUIRED), nil
		}
		args := request.Params.Arguments.(map[string]any)
		calendarID := args["calendar_id"].(string)
		eventID := args["event_id"].(string)

		event, err := calendarService.Events.Get(calendarID, eventID).Do()
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error getting event: %v", err)), nil
		}

		startTime := event.Start.DateTime
		if startTime == "" {
			startTime = event.Start.Date
		}

		endTime := event.End.DateTime
		if endTime == "" {
			endTime = event.End.Date
		}

		result := fmt.Sprintf(`Event Details:
Title: %s
Description: %s
Start: %s
End: %s
Location: %s
Status: %s
HTML Link: %s`,
			event.Summary,
			event.Description,
			startTime,
			endTime,
			event.Location,
			event.Status,
			event.HtmlLink)

		return mcp.NewToolResultText(result), nil
	})

	// Delete event tool
	deleteEventTool := mcp.NewTool("delete_event",
		mcp.WithDescription("Delete an event from Google Calendar"),
		mcp.WithString("calendar_id",
			mcp.Description("The calendar ID (use 'primary' for primary calendar)"),
			mcp.DefaultString("primary"),
		),
		mcp.WithString("event_id",
			mcp.Description("The event ID to delete"),
		),
	)

	s.AddTool(deleteEventTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if calendarService == nil {
			return mcp.NewToolResultError(TOOL_ERROR_AUTHENTICATION_REQUIRED), nil
		}
		args := request.Params.Arguments.(map[string]any)
		calendarID := args["calendar_id"].(string)
		eventID := args["event_id"].(string)

		err := calendarService.Events.Delete(calendarID, eventID).Do()
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error deleting event: %v", err)), nil
		}

		result := fmt.Sprintf("Event %s deleted successfully from calendar %s", eventID, calendarID)
		return mcp.NewToolResultText(result), nil
	})
}