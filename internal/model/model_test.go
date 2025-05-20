package model

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestParkingTicketMarshalUnmarshal tests that the ParkingTicket model can be
// properly marshaled to and unmarshaled from DynamoDB attributes
func TestParkingTicketMarshalUnmarshal(t *testing.T) {
	// Create a test ticket
	ticketID := uuid.New().String()
	plate := "ABC-123"
	parkingLot := 456
	entryTime := time.Now().UTC().Truncate(time.Millisecond) // Truncate to avoid precision issues

	ticket := &ParkingTicket{
		TicketID:   ticketID,
		Plate:      plate,
		ParkingLot: parkingLot,
		EntryTime:  entryTime,
	}

	// Marshal the ticket to DynamoDB attributes
	attrs, err := attributevalue.MarshalMap(ticket)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, attrs)

	// Check the marshaled attributes
	assert.Contains(t, attrs, "ticketId")
	assert.Contains(t, attrs, "plate")
	assert.Contains(t, attrs, "parkingLot")
	assert.Contains(t, attrs, "entryTime")

	// Unmarshal back to a ticket
	unmarshaled := &ParkingTicket{}
	err = attributevalue.UnmarshalMap(attrs, unmarshaled)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, ticketID, unmarshaled.TicketID)
	assert.Equal(t, plate, unmarshaled.Plate)
	assert.Equal(t, parkingLot, unmarshaled.ParkingLot)
	assert.Equal(t, entryTime, unmarshaled.EntryTime)
}
