// Package mocks provides mock implementations for testing
package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"parking-lot/internal/model"
)

// ParkingService is a mock implementation of the ParkingLotServicer interface
type ParkingService struct {
	mock.Mock
}

// CreateTicket mocks the ticket creation
func (m *ParkingService) CreateTicket(ctx context.Context, plate string, parkingLot int) (uuid.UUID, *model.ParkingTicket) {
	args := m.Called(ctx, plate, parkingLot)
	return args.Get(0).(uuid.UUID), args.Get(1).(*model.ParkingTicket)
}

// GetTicket mocks ticket retrieval
func (m *ParkingService) GetTicket(ctx context.Context, ticketID string) (*model.ParkingTicket, bool) {
	args := m.Called(ctx, ticketID)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*model.ParkingTicket), args.Bool(1)
}

// RemoveTicket mocks ticket removal
func (m *ParkingService) RemoveTicket(ctx context.Context, ticketID string) {
	m.Called(ctx, ticketID)
}

// CalculateCharge mocks charge calculation
func (m *ParkingService) CalculateCharge(entryTime time.Time) (int, float32) {
	args := m.Called(entryTime)
	return args.Int(0), args.Get(1).(float32)
}

// UpdateTicket mocks the ticket update
func (m *ParkingService) UpdateTicket(ctx context.Context, ticket *model.ParkingTicket) error {
	args := m.Called(ctx, ticket)
	return args.Error(0)
}
