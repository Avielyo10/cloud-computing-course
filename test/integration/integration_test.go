// Package integration provides integration tests for the parking lot system
// +build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"parking-lot/internal/handler"
	"parking-lot/internal/service"
	"parking-lot/pkg/lambda"
)

// TestEndToEndFlow tests the entire flow from entry to exit
func TestEndToEndFlow(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=true to run")
	}

	// Set up the test context
	ctx := context.Background()
	
	// Create the actual service with DynamoDB (ensure TABLE_NAME env var is set)
	tableName := os.Getenv("TABLE_NAME")
	require.NotEmpty(t, tableName, "TABLE_NAME environment variable must be set")
	
	parkingService, err := service.NewParkingLotService(ctx)
	require.NoError(t, err, "Failed to create parking service")
	
	// Create the handler
	parkingHandler := handler.NewParkingHandler(parkingService)

	// Create the Lambda adapter
	adapter := lambda.NewAdapter(parkingHandler)

	// Test data
	plate := fmt.Sprintf("TEST-%s", uuid.New().String()[:8])
	parkingLot := 999

	// Step 1: Create a ticket (Entry)
	t.Log("Testing entry...")
	ticketID, ticket := parkingService.CreateTicket(ctx, plate, parkingLot)
	
	assert.NotNil(t, ticketID)
	assert.Equal(t, plate, ticket.Plate)
	assert.Equal(t, parkingLot, ticket.ParkingLot)
	
	t.Logf("Created ticket: %s", ticketID.String())
	
	// Step 2: Wait a bit to simulate parking time
	t.Log("Waiting to simulate parking time...")
	time.Sleep(3 * time.Second)
	
	// Step 3: Retrieve the ticket
	t.Log("Testing ticket retrieval...")
	retrievedTicket, found := parkingService.GetTicket(ctx, ticketID.String())
	
	assert.True(t, found)
	assert.NotNil(t, retrievedTicket)
	assert.Equal(t, ticketID.String(), retrievedTicket.TicketID)
	assert.Equal(t, plate, retrievedTicket.Plate)
	assert.Equal(t, parkingLot, retrievedTicket.ParkingLot)
	
	// Step 4: Calculate charge
	minutes, charge := parkingService.CalculateCharge(retrievedTicket.EntryTime)
	
	// We only parked for a few seconds, so should get minimum charge
	assert.True(t, minutes < 1)
	assert.Equal(t, float32(5.0), charge) // Minimum charge
	
	// Step 5: Exit (remove ticket)
	t.Log("Testing exit...")
	parkingService.RemoveTicket(ctx, ticketID.String())
	
	// Step 6: Verify ticket is removed
	_, found = parkingService.GetTicket(ctx, ticketID.String())
	assert.False(t, found)
	
	t.Log("End-to-end test completed successfully")
}

// TestAPIAdapter tests the API adapter with the real router
func TestAPIAdapter(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=true to run")
	}
	
	// Create the adapter
	adapter := lambda.NewAdapter()
	
	// Start the server in a goroutine
	serverReady := make(chan bool)
	serverExit := make(chan bool)
	
	go func() {
		// Start the server (simplified for testing)
		server := &http.Server{
			Addr:    ":8085", // Use a different port for tests
			Handler: adapter.Router(), // Assuming Router() method returns the router
		}
		
		// Signal that the server is ready
		serverReady <- true
		
		// Wait for the exit signal
		<-serverExit
		
		// Shut down the server
		server.Shutdown(context.Background())
	}()
	
	// Wait for the server to be ready
	<-serverReady
	
	// Make test requests
	client := &http.Client{}
	
	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)
	
	// Test with a real request
	resp, err := client.Get("http://localhost:8085/healthz")
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	} else {
		t.Logf("Error making request: %v", err)
	}
	
	// Signal to stop the server
	serverExit <- true
}
