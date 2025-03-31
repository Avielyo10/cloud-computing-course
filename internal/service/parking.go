package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"parking-lot/internal/model"
)

// ParkingLotServicer defines the interface for parking lot operations
type ParkingLotServicer interface {
	// CreateTicket generates a new parking ticket
	CreateTicket(ctx context.Context, plate string, parkingLot int) (uuid.UUID, *model.ParkingTicket)

	// GetTicket retrieves a ticket by ID
	GetTicket(ctx context.Context, ticketID string) (*model.ParkingTicket, bool)

	// RemoveTicket removes a ticket from storage
	RemoveTicket(ctx context.Context, ticketID string)

	// CalculateCharge calculates parking fee
	CalculateCharge(entryTime time.Time) (int, float32)
}

// ParkingLotService handles parking lot operations with DynamoDB storage
type ParkingLotService struct {
	ctx       context.Context
	client    *dynamodb.Client
	tableName string
}

// NewParkingLotService creates a new service instance with DynamoDB
func NewParkingLotService(ctx context.Context) (*ParkingLotService, error) {
	// Get table name from environment variable
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		tableName = "parkingTickets" // Default table name
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create DynamoDB client
	client := dynamodb.NewFromConfig(cfg)

	return &ParkingLotService{
		ctx:       ctx,
		client:    client,
		tableName: tableName,
	}, nil
}

// CreateTicket generates a new parking ticket and stores it in DynamoDB
func (s *ParkingLotService) CreateTicket(ctx context.Context, plate string, parkingLot int) (uuid.UUID, *model.ParkingTicket) {
	// Generate a unique ticket ID
	ticketID := uuid.New()

	// Create the ticket
	ticket := &model.ParkingTicket{
		TicketID:   ticketID.String(),
		Plate:      plate,
		ParkingLot: parkingLot,
		EntryTime:  time.Now(),
	}

	// Marshal the ticket for DynamoDB
	item, err := attributevalue.MarshalMap(ticket)
	if err != nil {
		// Log error and return the ticket anyway (best effort)
		fmt.Printf("Error marshaling ticket: %v\n", err)
		return ticketID, ticket
	}

	// Store the ticket in DynamoDB
	_, err = s.client.PutItem(s.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		// Log error and return the ticket anyway (best effort)
		fmt.Printf("Error storing ticket in DynamoDB: %v\n", err)
	}

	return ticketID, ticket
}

// GetTicket retrieves a ticket by ID from DynamoDB
func (s *ParkingLotService) GetTicket(ctx context.Context, ticketID string) (*model.ParkingTicket, bool) {
	// Create the key for DynamoDB query
	key := map[string]types.AttributeValue{
		"TicketID": &types.AttributeValueMemberS{Value: ticketID},
	}

	// Get the item from DynamoDB
	result, err := s.client.GetItem(s.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key:       key,
	})
	if err != nil {
		fmt.Printf("Error retrieving ticket from DynamoDB: %v\n", err)
		return nil, false
	}

	// Check if item exists
	if result.Item == nil {
		return nil, false
	}

	// Unmarshal the item into a ticket
	ticket := &model.ParkingTicket{}
	if err := attributevalue.UnmarshalMap(result.Item, ticket); err != nil {
		fmt.Printf("Error unmarshaling ticket: %v\n", err)
		return nil, false
	}

	return ticket, true
}

// RemoveTicket removes a ticket from DynamoDB
func (s *ParkingLotService) RemoveTicket(ctx context.Context, ticketID string) {
	// Create the key for DynamoDB deletion
	key := map[string]types.AttributeValue{
		"TicketID": &types.AttributeValueMemberS{Value: ticketID},
	}

	// Delete the item from DynamoDB
	_, err := s.client.DeleteItem(s.ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key:       key,
	})
	if err != nil {
		fmt.Printf("Error deleting ticket from DynamoDB: %v\n", err)
	}
}

// CalculateCharge calculates parking fee
func (s *ParkingLotService) CalculateCharge(entryTime time.Time) (int, float32) {
	duration := time.Since(entryTime)
	minutes := int(duration.Minutes())

	// Calculate charge ($0.10 per minute with a minimum of $5)
	charge := float32(max(5.0, float64(minutes)*0.10))

	return minutes, charge
}

// max returns the larger of x or y
func max(x, y float64) float64 {
	if x > y {
		return x
	}
	return y
}
