package model

import (
	"time"
)

// TicketStatus represents the status of a parking ticket.
// +enum
type TicketStatus string

const (
	// TicketStatusIn indicates the vehicle is in the parking lot.
	TicketStatusIn TicketStatus = "in"
	// TicketStatusOut indicates the vehicle has exited the parking lot.
	TicketStatusOut TicketStatus = "out"
)

// ParkingTicket represents a parking session
type ParkingTicket struct {
	TicketID   string       `dynamodbav:"ticketId" json:"ticketId"`
	Plate      string       `dynamodbav:"plate" json:"plate"`
	ParkingLot int          `dynamodbav:"parkingLot" json:"parkingLot"`
	EntryTime  time.Time    `dynamodbav:"entryTime" json:"entryTime"`
	Status     TicketStatus `dynamodbav:"status,omitempty" json:"status,omitempty"`
	Charge     float32      `dynamodbav:"charge,omitempty" json:"charge,omitempty"`
}
