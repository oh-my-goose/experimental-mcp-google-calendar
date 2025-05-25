package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var (
	calendarService *calendar.Service
)

func main() {
	// // Initialize Google Calendar service
	// if err := initCalendarService(); err != nil {
	// 	log.Fatalf("Failed to initialize Calendar service: %v", err)
	// }

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

	// Start the server
	// if err := server.ServeStdio(s); err != nil {
	// 	fmt.Printf("Error starting server: %v\n", err)
	// }

	host := "localhost"
	port := "12345"

	sseServer := server.NewSSEServer(mcpServer,
		server.WithBaseURL(fmt.Sprintf("http://%s:%s", host, port)),
		server.WithStaticBasePath("/mcp"),
	)

	mux := http.NewServeMux()
	mux.Handle("/mcp/sse", sseServer.SSEHandler())
	mux.Handle("/mcp/message", sseServer.MessageHandler())

	log.Printf("Server listening at http://%s:%s", host, port)
	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", host, port), mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func initCalendarService() error {
	ctx := context.Background()

	// Read credentials file
	credentialsFile := os.Getenv("GOOGLE_CREDENTIALS_FILE")
	if credentialsFile == "" {
		credentialsFile = "credentials.json"
	}

	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		return fmt.Errorf("unable to read client secret file: %v", err)
	}

	// Configure OAuth2
	config, err := google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		return fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	client := getClient(config)

	// Create Calendar service
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("unable to retrieve Calendar client: %v", err)
	}

	calendarService = srv
	return nil
}

func getClient(config *oauth2.Config) *http.Client {
	// Try to load token from file
	tokenFile := "token.json"
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func setupTools(s *server.MCPServer) {
	// List calendars tool
	listCalendarsTool := mcp.NewTool("list_calendars",
		mcp.WithDescription("List all accessible Google Calendars"),
	)

	s.AddTool(listCalendarsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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