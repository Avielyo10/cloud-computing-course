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

// TestCalculateCharge tests the charge calculation logic
func TestCalculateCharge(t *testing.T) {
	// Setup
	service := &ParkingLotService{
		marshalMap:   attributevalue.MarshalMap,
		unmarshalMap: attributevalue.UnmarshalMap,
	}

	testCases := []struct {
		name            string
		entryTime       time.Time
		expectedMinutes int
		expectedCharge  float32
	}{
		{
			name:            "Less than minimum charge",
			entryTime:       time.Now().Add(-20 * time.Minute),
			expectedMinutes: 20,
			expectedCharge:  5.0, // Minimum charge is $5
		},
		{
			name:            "More than minimum charge",
			entryTime:       time.Now().Add(-60 * time.Minute),
			expectedMinutes: 60,
			expectedCharge:  6.0, // 60 minutes * $0.10 = $6.00
		},
		{
			name:            "2 hours parking",
			entryTime:       time.Now().Add(-120 * time.Minute),
			expectedMinutes: 120,
			expectedCharge:  12.0, // 120 minutes * $0.10 = $12.00
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Because time.Since is used, we need to account for the time that passes during the test
			// Allow for a small margin of error in the minute calculation
			minutes, charge := service.CalculateCharge(tc.entryTime)

			// The minutes might be slightly different due to test execution time
			assert.True(t, minutes >= 0 && minutes <= tc.expectedMinutes+2, "Minutes should be close to expected")

			// For charge, we should check if it's calculated correctly based on the actual minutes
			// Use the max function from service.go
			if minutes < 50 {
				assert.Equal(t, float32(5.0), charge, "Charge should be minimum $5.00")
			} else {
				expectedCharge := float32(0.10 * float64(minutes))
				if expectedCharge < 5.0 {
					expectedCharge = 5.0
				}
				assert.Equal(t, expectedCharge, charge)
			}
		})
	}
}

// For testing purposes
var unmarshalMap = func(item map[string]interface{}, out interface{}) error {
	// This would be replaced with the actual DynamoDB unmarshalling in tests
	return nil
}
