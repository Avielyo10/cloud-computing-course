package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"parking-lot/internal/mocks"
	"parking-lot/internal/model"
	"parking-lot/server/api"
)

// MockParkingService is a mock implementation of the ParkingLotServicer interface
type MockParkingService struct {
	mock.Mock
}

// CreateTicket mocks the ticket creation
func (m *MockParkingService) CreateTicket(ctx context.Context, plate string, parkingLot int) (uuid.UUID, *model.ParkingTicket) {
	args := m.Called(ctx, plate, parkingLot)
	return args.Get(0).(uuid.UUID), args.Get(1).(*model.ParkingTicket)
}

// GetTicket mocks ticket retrieval
func (m *MockParkingService) GetTicket(ctx context.Context, ticketID string) (*model.ParkingTicket, bool) {
	args := m.Called(ctx, ticketID)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*model.ParkingTicket), args.Bool(1)
}

// RemoveTicket mocks ticket removal
func (m *MockParkingService) RemoveTicket(ctx context.Context, ticketID string) {
	m.Called(ctx, ticketID)
}

// CalculateCharge mocks charge calculation
func (m *MockParkingService) CalculateCharge(entryTime time.Time) (int, float32) {
	args := m.Called(entryTime)
	return args.Int(0), args.Get(1).(float32)
}

// setupTestRouter creates a router with the handler for testing
func setupTestRouter(mockService *mocks.ParkingService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewParkingHandler(mockService)

	router.POST("/entry", func(c *gin.Context) {
		params := api.PostEntryParams{
			Plate:      c.Query("plate"),
			ParkingLot: 0, // Will be set from query
		}

		// Parse the parkingLot parameter
		if parkingLot := c.Query("parkingLot"); parkingLot != "" {
			var lot int
			_, err := fmt.Sscanf(parkingLot, "%d", &lot)
			if err == nil {
				params.ParkingLot = lot
			}
		}

		handler.PostEntry(c, params)
	})

	router.POST("/exit", func(c *gin.Context) {
		params := api.PostExitParams{}
		ticketIdStr := c.Query("ticketId")
		if ticketIdStr != "" {
			ticketId, err := uuid.Parse(ticketIdStr)
			if err == nil {
				params.TicketId = ticketId
			}
		}
		handler.PostExit(c, params)
	})

	return router
}

// TestPostEntry tests the entry handler functionality
func TestPostEntry(t *testing.T) {
	// Setup mock service
	mockService := new(mocks.ParkingService)
	router := setupTestRouter(mockService)

	// Test data
	testPlate := "ABC-123"
	testParkingLot := 456
	testTicketID := uuid.New()
	testTicket := &model.ParkingTicket{
		TicketID:   testTicketID.String(),
		Plate:      testPlate,
		ParkingLot: testParkingLot,
		EntryTime:  time.Now(),
	}

	// Setup expectations
	mockService.On("CreateTicket", mock.Anything, testPlate, testParkingLot).Return(testTicketID, testTicket)

	// Create test request
	req := httptest.NewRequest("POST", "/entry?plate="+testPlate+"&parkingLot="+strconv.Itoa(testParkingLot), nil)
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var response api.EntryResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, testTicketID, response.TicketId)

	// Verify mock expectations
	mockService.AssertExpectations(t)
}

// TestPostExit tests the exit handler functionality
func TestPostExit(t *testing.T) {
	// Setup mock service
	mockService := new(mocks.ParkingService)
	router := setupTestRouter(mockService)

	// Common test data
	testTicketID := uuid.New()
	testPlate := "XYZ-789"
	testParkingLot := 123
	testEntryTime := time.Now().Add(-45 * time.Minute)

	testTicket := &model.ParkingTicket{
		TicketID:   testTicketID.String(),
		Plate:      testPlate,
		ParkingLot: testParkingLot,
		EntryTime:  testEntryTime,
	}

	// Test case: Successful exit
	t.Run("Successful exit", func(t *testing.T) {
		// Setup expectations for successful exit
		mockService.On("GetTicket", mock.Anything, testTicketID.String()).Return(testTicket, true).Once()
		mockService.On("CalculateCharge", testEntryTime).Return(45, float32(5.0)).Once()
		mockService.On("RemoveTicket", mock.Anything, testTicketID.String()).Once()

		// Create test request
		req := httptest.NewRequest("POST", "/exit?ticketId="+testTicketID.String(), nil)
		w := httptest.NewRecorder()

		// Perform the request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		// Parse response
		var response api.ExitResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, testPlate, response.Plate)
		assert.Equal(t, testParkingLot, response.ParkingLot)
		assert.Equal(t, 45, response.ParkedDurationMinutes)
		assert.Equal(t, float32(5.0), response.Charge)

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	// Test case: Ticket not found
	t.Run("Ticket not found", func(t *testing.T) {
		// Reset mock
		mockService.ExpectedCalls = nil

		// Setup expectations for ticket not found
		nonExistentTicketID := uuid.New()
		mockService.On("GetTicket", mock.Anything, nonExistentTicketID.String()).Return(nil, false).Once()

		// Create test request
		req := httptest.NewRequest("POST", "/exit?ticketId="+nonExistentTicketID.String(), nil)
		w := httptest.NewRecorder()

		// Perform the request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusNotFound, w.Code)

		// Parse response
		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, "Ticket not found", response.Message)

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})
}
