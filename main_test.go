package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Always call cancel to release resources

	go func() {
		// Simulate some work
		time.Sleep(6 * time.Second)
		cancel() // Cancel the context after 2 seconds
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Context was canceled!")
	case <-time.After(5 * time.Second):
		fmt.Println("Time out!")
		cancel()
	}
}

func TestDeadline(t *testing.T) {
	deadline := time.Now().Add(3 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	select {
	case <-ctx.Done():
		fmt.Println("Context deadline reached!")
	case <-time.After(5 * time.Second):
		fmt.Println("Work completed")
	}
}

// func TestAuthTool(t *testing.T) {
// 	// Create a test MCP server
// 	mcpServer := server.NewMCPServer(
// 		"Google Calendar MCP",
// 		"0.1.0",
// 		server.WithToolCapabilities(false),
// 	)

// 	// Setup test OAuth config
// 	oauthConfig = &oauth2.Config{
// 		ClientID:     "test-client-id",
// 		ClientSecret: "test-client-secret",
// 		RedirectURL:  "http://localhost:5555/auth/callback",
// 		Scopes: []string{
// 			"https://www.googleapis.com/auth/userinfo.email",
// 			"https://www.googleapis.com/auth/userinfo.profile",
// 			"https://www.googleapis.com/auth/calendar.readonly",
// 			"openid",
// 		},
// 		Endpoint: oauth2.Endpoint{
// 			AuthURL:  "https://test.com/auth",
// 			TokenURL: "https://test.com/token",
// 		},
// 	}

// 	// Setup auth tool
// 	authTool := mcp.NewTool("auth",
// 		mcp.WithDescription("Authenticate with Google Calendar to use the other tools."),
// 	)

// 	mcpServer.AddTool(authTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 		url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
// 		return mcp.NewToolResultText("Please visit this URL to authenticate: " + url), nil
// 	})

// 	// Create test request
// 	req := httptest.NewRequest("POST", "/mcp/message", nil)
// 	w := httptest.NewRecorder()

// 	// Create SSE server and handle request
// 	sseServer := server.NewSSEServer(mcpServer,
// 		server.WithBaseURL("http://localhost:5555"),
// 		server.WithStaticBasePath("/mcp"),
// 	)

// 	sseServer.MessageHandler().ServeHTTP(w, req)

// 	// Verify response
// 	if w.Code != http.StatusOK {
// 		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
// 	}
// }

// func TestListCalendarsTool(t *testing.T) {
// 	// Create a test MCP server
// 	mcpServer := server.NewMCPServer(
// 		"Google Calendar MCP",
// 		"0.1.0",
// 		server.WithToolCapabilities(false),
// 	)

// 	// Setup test calendar service
// 	calendarService = &calendar.Service{}

// 	// Setup list calendars tool
// 	listCalendarsTool := mcp.NewTool("list_calendars",
// 		mcp.WithDescription("List all accessible Google Calendars"),
// 	)

// 	mcpServer.AddTool(listCalendarsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 		return mcp.NewToolResultText("Available Calendars:\n- Test Calendar (ID: test-id)"), nil
// 	})

// 	// Create test request
// 	req := httptest.NewRequest("POST", "/mcp/message", nil)
// 	w := httptest.NewRecorder()

// 	// Create SSE server and handle request
// 	sseServer := server.NewSSEServer(mcpServer,
// 		server.WithBaseURL("http://localhost:5555"),
// 		server.WithStaticBasePath("/mcp"),
// 	)

// 	sseServer.MessageHandler().ServeHTTP(w, req)

// 	// Verify response
// 	if w.Code != http.StatusOK {
// 		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
// 	}
// }

// func TestListEventsTool(t *testing.T) {
// 	// Create a test MCP server
// 	mcpServer := server.NewMCPServer(
// 		"Google Calendar MCP",
// 		"0.1.0",
// 		server.WithToolCapabilities(false),
// 	)

// 	// Setup test calendar service
// 	calendarService = &calendar.Service{}

// 	// Setup list events tool
// 	listEventsTool := mcp.NewTool("list_events",
// 		mcp.WithDescription("List events from a Google Calendar"),
// 		mcp.WithString("calendar_id",
// 			mcp.Description("The calendar ID"),
// 			mcp.DefaultString("primary"),
// 		),
// 		mcp.WithNumber("max_results",
// 			mcp.Description("Maximum number of events to return"),
// 			mcp.DefaultNumber(10),
// 		),
// 	)

// 	mcpServer.AddTool(listEventsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 		return mcp.NewToolResultText("Events in calendar primary:\n- Test Event (2024-01-01T10:00:00Z)"), nil
// 	})

// 	// Create test request
// 	req := httptest.NewRequest("POST", "/mcp/message", nil)
// 	w := httptest.NewRecorder()

// 	// Create SSE server and handle request
// 	sseServer := server.NewSSEServer(mcpServer,
// 		server.WithBaseURL("http://localhost:5555"),
// 		server.WithStaticBasePath("/mcp"),
// 	)

// 	sseServer.MessageHandler().ServeHTTP(w, req)

// 	// Verify response
// 	if w.Code != http.StatusOK {
// 		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
// 	}
// }

// func TestCreateEventTool(t *testing.T) {
// 	// Create a test MCP server
// 	mcpServer := server.NewMCPServer(
// 		"Google Calendar MCP",
// 		"0.1.0",
// 		server.WithToolCapabilities(false),
// 	)

// 	// Setup test calendar service
// 	calendarService = &calendar.Service{}

// 	// Setup create event tool
// 	createEventTool := mcp.NewTool("create_event",
// 		mcp.WithDescription("Create a new event in Google Calendar"),
// 		mcp.WithString("calendar_id",
// 			mcp.Description("The calendar ID"),
// 			mcp.DefaultString("primary"),
// 		),
// 		mcp.WithString("summary",
// 			mcp.Description("Event title/summary"),
// 		),
// 		mcp.WithString("start_time",
// 			mcp.Description("Event start time"),
// 		),
// 		mcp.WithString("end_time",
// 			mcp.Description("Event end time"),
// 		),
// 	)

// 	mcpServer.AddTool(createEventTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 		return mcp.NewToolResultText("Event created successfully!\nTitle: Test Event\nID: test-id\nHTML Link: http://test.com"), nil
// 	})

// 	// Create test request
// 	req := httptest.NewRequest("POST", "/mcp/message", nil)
// 	w := httptest.NewRecorder()

// 	// Create SSE server and handle request
// 	sseServer := server.NewSSEServer(mcpServer,
// 		server.WithBaseURL("http://localhost:5555"),
// 		server.WithStaticBasePath("/mcp"),
// 	)

// 	sseServer.MessageHandler().ServeHTTP(w, req)

// 	// Verify response
// 	if w.Code != http.StatusOK {
// 		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
// 	}
// }

// func TestGetEventTool(t *testing.T) {
// 	// Create a test MCP server
// 	mcpServer := server.NewMCPServer(
// 		"Google Calendar MCP",
// 		"0.1.0",
// 		server.WithToolCapabilities(false),
// 	)

// 	// Setup test calendar service
// 	calendarService = &calendar.Service{}

// 	// Setup get event tool
// 	getEventTool := mcp.NewTool("get_event",
// 		mcp.WithDescription("Get details of a specific event"),
// 		mcp.WithString("calendar_id",
// 			mcp.Description("The calendar ID"),
// 			mcp.DefaultString("primary"),
// 		),
// 		mcp.WithString("event_id",
// 			mcp.Description("The event ID"),
// 		),
// 	)

// 	mcpServer.AddTool(getEventTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 		return mcp.NewToolResultText("Event Details:\nTitle: Test Event\nID: test-id\nStart: 2024-01-01T10:00:00Z\nEnd: 2024-01-01T11:00:00Z"), nil
// 	})

// 	// Create test request
// 	req := httptest.NewRequest("POST", "/mcp/message", nil)
// 	w := httptest.NewRecorder()

// 	// Create SSE server and handle request
// 	sseServer := server.NewSSEServer(mcpServer,
// 		server.WithBaseURL("http://localhost:5555"),
// 		server.WithStaticBasePath("/mcp"),
// 	)

// 	sseServer.MessageHandler().ServeHTTP(w, req)

// 	// Verify response
// 	if w.Code != http.StatusOK {
// 		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
// 	}
// }
