package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"parking-lot/internal/logger"
	"parking-lot/internal/mocks"
	"parking-lot/internal/model"
)

// TestCreateTicket tests the ticket creation functionality
func TestCreateTicket(t *testing.T) {
	// Setup
	ctx := context.Background()
	service := &ParkingLotService{
		ctx:          ctx,
		client:       &mocks.DynamoDBClient{},
		tableName:    "testTable",
		log:          logger.NewLogger(),
		marshalMap:   attributevalue.MarshalMap,
		unmarshalMap: attributevalue.UnmarshalMap,
	}

	// Expected values
	plate := "ABC-123"
	parkingLot := 123

	service.client.(*mocks.DynamoDBClient).On("PutItem", ctx, mock.Anything, mock.Anything).Return(&dynamodb.PutItemOutput{}, nil).Once()

	// Call the function
	ticketID, ticket := service.CreateTicket(ctx, plate, parkingLot)

	// Assertions
	assert.NotNil(t, ticketID)
	assert.NotEmpty(t, ticket.TicketID)
	assert.Equal(t, plate, ticket.Plate)
	assert.Equal(t, parkingLot, ticket.ParkingLot)
	assert.WithinDuration(t, time.Now(), ticket.EntryTime, 2*time.Second)
	assert.Equal(t, model.TicketStatusIn, ticket.Status)
	assert.Equal(t, float32(0.0), ticket.Charge)

	service.client.(*mocks.DynamoDBClient).AssertExpectations(t)
}

// TestCreateTicket_MarshalError tests the ticket creation with Marshal error
func TestCreateTicket_MarshalError(t *testing.T) {
	ctx := context.Background()
	mockClient := new(mocks.DynamoDBClient)
	service := &ParkingLotService{
		ctx:       ctx,
		client:    mockClient,
		tableName: "testTable",
		log:       logger.NewLogger(),
		marshalMap: func(interface{}) (map[string]types.AttributeValue, error) {
			return nil, fmt.Errorf("marshal error")
		},
		unmarshalMap: attributevalue.UnmarshalMap,
	}
	id, ticket := service.CreateTicket(ctx, "PLATE", 1)
	assert.NotNil(t, id)
	assert.NotNil(t, ticket)
}

// TestCreateTicket_PutItemError tests the ticket creation with PutItem error
func TestCreateTicket_PutItemError(t *testing.T) {
	ctx := context.Background()
	mockClient := new(mocks.DynamoDBClient)
	service := &ParkingLotService{
		ctx:          ctx,
		client:       mockClient,
		tableName:    "testTable",
		log:          logger.NewLogger(),
		marshalMap:   attributevalue.MarshalMap,
		unmarshalMap: attributevalue.UnmarshalMap,
	}
	mockClient.On("PutItem", ctx, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("put error"))
	id, ticket := service.CreateTicket(ctx, "PLATE", 1)
	assert.NotNil(t, id)
	assert.NotNil(t, ticket)
}

// TestGetTicket tests retrieving a ticket
func TestGetTicket(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockClient := new(mocks.DynamoDBClient)
	service := &ParkingLotService{
		ctx:          ctx,
		client:       mockClient,
		tableName:    "testTable",
		log:          logger.NewLogger(),
		marshalMap:   attributevalue.MarshalMap,
		unmarshalMap: attributevalue.UnmarshalMap,
	}

	// Test data
	testTicketID := uuid.New().String()
	testPlate := "XYZ-789"
	testParkingLot := 456
	testEntryTime := time.Now().Add(-30 * time.Minute)

	// Create a test ticket
	testTicket := &model.ParkingTicket{
		TicketID:   testTicketID,
		Plate:      testPlate,
		ParkingLot: testParkingLot,
		EntryTime:  testEntryTime,
	}
	// Test case: Ticket found
	t.Run("Ticket found", func(t *testing.T) {
		// Marshal the test ticket to a DynamoDB item
		item, err := attributevalue.MarshalMap(testTicket)
		assert.NoError(t, err)

		mockClient.ExpectedCalls = nil // Reset mock
		mockClient.On("GetItem", ctx, mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{Item: item}, nil).Once()

		// Call the function
		ticket, found := service.GetTicket(ctx, testTicketID)

		// Assertions
		assert.True(t, found)
		assert.Equal(t, testTicket.TicketID, ticket.TicketID)
		assert.Equal(t, testPlate, ticket.Plate)
		assert.Equal(t, testParkingLot, ticket.ParkingLot)
		assert.WithinDuration(t, testEntryTime, ticket.EntryTime, 2*time.Second)

		// Verify mock
		mockClient.AssertCalled(t, "GetItem", ctx, mock.Anything, mock.Anything)
	})

	// Test case: Ticket not found
	t.Run("Ticket not found", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset mock
		mockClient.On("GetItem", ctx, mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{Item: nil}, nil).Once()

		// Call the function
		ticket, found := service.GetTicket(ctx, "non-existent-id")

		// Assertions
		assert.False(t, found)
		assert.Nil(t, ticket)

		// Verify mock
		mockClient.AssertCalled(t, "GetItem", ctx, mock.Anything, mock.Anything)
	})
}

// TestGetTicket_GetItemError tests error handling in GetTicket
func TestGetTicket_GetItemError(t *testing.T) {
	ctx := context.Background()
	mockClient := new(mocks.DynamoDBClient)
	service := &ParkingLotService{
		ctx:          ctx,
		client:       mockClient,
		tableName:    "testTable",
		log:          logger.NewLogger(),
		marshalMap:   attributevalue.MarshalMap,
		unmarshalMap: attributevalue.UnmarshalMap,
	}
	mockClient.On("GetItem", ctx, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("get error")).Once()
	ticket, found := service.GetTicket(ctx, "id")
	assert.False(t, found)
	assert.Nil(t, ticket)
}

// TestGetTicket_UnmarshalError tests error handling in GetTicket when unmarshalling
func TestGetTicket_UnmarshalError(t *testing.T) {
	ctx := context.Background()
	mockClient := new(mocks.DynamoDBClient)
	service := &ParkingLotService{
		ctx:        ctx,
		client:     mockClient,
		tableName:  "testTable",
		log:        logger.NewLogger(),
		marshalMap: attributevalue.MarshalMap,
		unmarshalMap: func(map[string]types.AttributeValue, interface{}) error {
			return fmt.Errorf("unmarshal error")
		},
	}
	item := map[string]types.AttributeValue{"TicketID": &types.AttributeValueMemberS{Value: "id"}}
	mockClient.On("GetItem", ctx, mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{Item: item}, nil).Once()
	ticket, found := service.GetTicket(ctx, "id")
	assert.False(t, found)
	assert.Nil(t, ticket)
}

// TestRemoveTicket tests the ticket removal functionality
func TestRemoveTicket(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockClient := new(mocks.DynamoDBClient)
	service := &ParkingLotService{
		ctx:          ctx,
		client:       mockClient,
		tableName:    "testTable",
		log:          logger.NewLogger(),
		marshalMap:   attributevalue.MarshalMap,
		unmarshalMap: attributevalue.UnmarshalMap,
	}

	testTicketID := uuid.New().String()

	// Mock the DynamoDB DeleteItem response
	mockClient.On("DeleteItem", mock.Anything, mock.Anything, mock.Anything).Return(&dynamodb.DeleteItemOutput{}, nil)

	// Call the function
	service.RemoveTicket(ctx, testTicketID)

	// Verify that DeleteItem was called with correct parameters
	mockClient.AssertCalled(t, "DeleteItem", ctx, mock.Anything, mock.Anything)
}

// TestRemoveTicket_DeleteItemError tests error handling in RemoveTicket
func TestRemoveTicket_DeleteItemError(t *testing.T) {
	ctx := context.Background()
	mockClient := new(mocks.DynamoDBClient)
	service := &ParkingLotService{
		ctx:          ctx,
		client:       mockClient,
		tableName:    "testTable",
		log:          logger.NewLogger(),
		marshalMap:   attributevalue.MarshalMap,
		unmarshalMap: attributevalue.UnmarshalMap,
	}
	mockClient.On("DeleteItem", ctx, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("delete error")).Once()
	service.RemoveTicket(ctx, "id")
}

// TestUpdateTicket tests updating a ticket
func TestUpdateTicket(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockClient := new(mocks.DynamoDBClient)
	service := &ParkingLotService{
		ctx:          ctx,
		client:       mockClient,
		tableName:    "testTable",
		log:          logger.NewLogger(),
		marshalMap:   attributevalue.MarshalMap,
		unmarshalMap: attributevalue.UnmarshalMap,
	}

	// Test data
	testTicket := &model.ParkingTicket{
		TicketID:   uuid.New().String(),
		Plate:      "UPD-001",
		ParkingLot: 789,
		EntryTime:  time.Now().Add(-60 * time.Minute),
		Status:     model.TicketStatusOut,
		Charge:     10.0,
	}

	// Mock the DynamoDB PutItem response for update
	mockClient.On("PutItem", ctx, mock.AnythingOfType("*dynamodb.PutItemInput"), mock.Anything).Return(&dynamodb.PutItemOutput{}, nil).Once()

	// Call the function
	err := service.UpdateTicket(ctx, testTicket)

	// Assertions
	assert.NoError(t, err)

	// Verify that PutItem was called
	mockClient.AssertCalled(t, "PutItem", ctx, mock.AnythingOfType("*dynamodb.PutItemInput"), mock.Anything)
}

// TestUpdateTicket_MarshalError tests error handling in UpdateTicket when marshalling
func TestUpdateTicket_MarshalError(t *testing.T) {
	ctx := context.Background()
	mockClient := new(mocks.DynamoDBClient) // No need to set expectations if marshal fails before client call
	service := &ParkingLotService{
		ctx:       ctx,
		client:    mockClient,
		tableName: "testTable",
		log:       logger.NewLogger(),
		marshalMap: func(in interface{}) (map[string]types.AttributeValue, error) {
			return nil, fmt.Errorf("marshal error")
		},
		unmarshalMap: attributevalue.UnmarshalMap,
	}

	testTicket := &model.ParkingTicket{TicketID: "test-id"} // Minimal ticket for the test
	err := service.UpdateTicket(ctx, testTicket)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal error")
}

// TestUpdateTicket_PutItemError tests error handling in UpdateTicket when PutItem fails
func TestUpdateTicket_PutItemError(t *testing.T) {
	ctx := context.Background()
	mockClient := new(mocks.DynamoDBClient)
	service := &ParkingLotService{
		ctx:          ctx,
		client:       mockClient,
		tableName:    "testTable",
		log:          logger.NewLogger(),
		marshalMap:   attributevalue.MarshalMap,
		unmarshalMap: attributevalue.UnmarshalMap,
	}

	testTicket := &model.ParkingTicket{TicketID: "test-id"} // Minimal ticket for the test

	mockClient.On("PutItem", ctx, mock.AnythingOfType("*dynamodb.PutItemInput"), mock.Anything).Return(nil, fmt.Errorf("put error")).Once()

	err := service.UpdateTicket(ctx, testTicket)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "put error")
	mockClient.AssertCalled(t, "PutItem", ctx, mock.AnythingOfType("*dynamodb.PutItemInput"), mock.Anything)
}

// TestCalculateCharge tests the charge calculation logic
func TestCalculateCharge(t *testing.T) {
	// Setup
	service := &ParkingLotService{
		// No need for marshalMap/unmarshalMap for this specific test
		// as CalculateCharge doesn't interact with DynamoDB.
		// log: logger.NewLogger(), // Initialize logger if CalculateCharge uses it (it doesn't currently)
	}

	testCases := []struct {
		name            string
		duration        time.Duration // Use duration for more precise control
		expectedMinutes int
		expectedCharge  float32
	}{
		{
			name:            "0 minutes (edge case, should be 0 charge)",
			duration:        0 * time.Minute,
			expectedMinutes: 0,
			expectedCharge:  0.0, // Correct: 0 increments
		},
		{
			name:            "1 minute (1st 15-min increment)",
			duration:        1 * time.Minute,
			expectedMinutes: 1,
			expectedCharge:  2.50, // Correct: 1 increment * $2.50
		},
		{
			name:            "14.999 minutes (1st 15-min increment)",
			duration:        14*time.Minute + 59*time.Second + 999*time.Millisecond,
			expectedMinutes: 14,   // approx
			expectedCharge:  2.50, // Correct: 1 increment * $2.50
		},
		{
			name:            "15 minutes (1st 15-min increment)",
			duration:        15 * time.Minute,
			expectedMinutes: 15,
			expectedCharge:  2.50, // Correct: 1 increment * $2.50
		},
		{
			name:            "15.001 minutes (2nd 15-min increment)", // Barely into the 2nd increment
			duration:        15*time.Minute + 1*time.Millisecond,
			expectedMinutes: 15,   // approx
			expectedCharge:  5.00, // Correct: 2 increments * $2.50
		},
		{
			name:            "16 minutes (2nd 15-min increment)",
			duration:        16 * time.Minute,
			expectedMinutes: 16,
			expectedCharge:  5.00, // Correct: 2 increments * $2.50
		},
		{
			name:            "29.999 minutes (2nd 15-min increment)",
			duration:        29*time.Minute + 59*time.Second + 999*time.Millisecond,
			expectedMinutes: 29,   // approx
			expectedCharge:  5.00, // Correct: 2 increments * $2.50
		},
		{
			name:            "30 minutes (2nd 15-min increment)",
			duration:        30 * time.Minute,
			expectedMinutes: 30,
			expectedCharge:  5.00, // Correct: 2 increments * $2.50
		},
		{
			name:            "30.001 minutes (3rd 15-min increment)",
			duration:        30*time.Minute + 1*time.Millisecond,
			expectedMinutes: 30,   // approx
			expectedCharge:  7.50, // Correct: 3 increments * $2.50
		},
		{
			name:            "50 minutes (4th 15-min increment)", // ceil(50/15) = 4
			duration:        50 * time.Minute,
			expectedMinutes: 50,
			expectedCharge:  10.00, // Correct: 4 increments * $2.50
		},
		{
			name:            "59.999 minutes (4th 15-min increment)",
			duration:        59*time.Minute + 59*time.Second + 999*time.Millisecond,
			expectedMinutes: 59,    // approx
			expectedCharge:  10.00, // Correct: 4 increments * $2.50
		},
		{
			name:            "60 minutes / 1 hour (4th 15-min increment)", // ceil(60/15) = 4
			duration:        60 * time.Minute,
			expectedMinutes: 60,
			expectedCharge:  10.00, // Correct: 4 increments * $2.50
		},
		{
			name:            "60.001 minutes (5th 15-min increment)",
			duration:        60*time.Minute + 1*time.Millisecond,
			expectedMinutes: 60,    // approx
			expectedCharge:  12.50, // Correct: 5 increments * $2.50
		},
		{
			name:            "70 minutes (5th 15-min increment)", // ceil(70/15) = 5
			duration:        70 * time.Minute,
			expectedMinutes: 70,
			expectedCharge:  12.50, // Correct: 5 increments * $2.50
		},
		{
			name:            "119.999 minutes (8th 15-min increment)",
			duration:        119*time.Minute + 59*time.Second + 999*time.Millisecond,
			expectedMinutes: 119,   // approx
			expectedCharge:  20.00, // Correct: 8 increments * $2.50
		},
		{
			name:            "120 minutes / 2 hours (8th 15-min increment)", // ceil(120/15) = 8
			duration:        120 * time.Minute,
			expectedMinutes: 120,
			expectedCharge:  20.00, // Correct: 8 increments * $2.50
		},
		{
			name:            "120.001 minutes (9th 15-min increment)",
			duration:        120*time.Minute + 1*time.Millisecond,
			expectedMinutes: 120,   // approx
			expectedCharge:  22.50, // Correct: 9 increments * $2.50
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the entry time by subtracting the duration from the current time
			entryTime := time.Now().Add(-tc.duration)

			minutes, charge := service.CalculateCharge(entryTime)

			// Allow for a small discrepancy in minutes due to test execution time.
			// The actual minutes calculated by time.Since(entryTime) might be slightly
			// greater than tc.expectedMinutes.
			// We check if 'minutes' is very close to 'tc.expectedMinutes'.
			// A common way is to check if it's within a small delta, e.g., expectedMinutes or expectedMinutes + 1.
			// For durations like 0, it should be 0 or 1.
			assert.InDelta(t, tc.expectedMinutes, minutes, 1.5, "Minutes should be very close to expected")

			assert.Equal(t, tc.expectedCharge, charge, "Charge should match expected value")
		})
	}
}

// For testing purposes
var unmarshalMap = func(item map[string]interface{}, out interface{}) error {
	// This would be replaced with the actual DynamoDB unmarshalling in tests
	return nil
}
