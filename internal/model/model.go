package model

import (
	"time"
)

// ParkingTicket represents a parking session
type ParkingTicket struct {
	TicketID   string    `dynamodbav:"ticketId" json:"ticketId"`
	Plate      string    `dynamodbav:"plate" json:"plate"`
	ParkingLot int       `dynamodbav:"parkingLot" json:"parkingLot"`
	EntryTime  time.Time `dynamodbav:"entryTime" json:"entryTime"`
}
