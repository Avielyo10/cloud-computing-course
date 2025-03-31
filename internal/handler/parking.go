package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"parking-lot/internal/service"
	"parking-lot/server/api"
)

// ParkingHandler implements the ServerInterface
type ParkingHandler struct {
	service service.ParkingLotServicer
}

// NewParkingHandler creates a new handler with the given service
func NewParkingHandler(service service.ParkingLotServicer) *ParkingHandler {
	return &ParkingHandler{service: service}
}

// PostEntry records a vehicle entry and generates a ticket
func (h *ParkingHandler) PostEntry(c *gin.Context, params api.PostEntryParams) {
	ticketID, _ := h.service.CreateTicket(c, params.Plate, params.ParkingLot)

	// Return the ticket ID
	response := api.EntryResponse{
		TicketId: &ticketID,
	}
	c.JSON(http.StatusOK, response)
}

// PostExit processes a vehicle exit
func (h *ParkingHandler) PostExit(c *gin.Context, params api.PostExitParams) {
	ticket, exists := h.service.GetTicket(c, params.TicketId)
	if !exists {
		errorMsg := "Ticket not found"
		response := api.ErrorResponse{
			Message: &errorMsg,
		}
		c.JSON(http.StatusNotFound, response)
		return
	}

	// Calculate parking duration and charge
	minutes, charge := h.service.CalculateCharge(ticket.EntryTime)

	// Create response
	response := api.ExitResponse{
		Plate:                 &ticket.Plate,
		ParkingLot:            &ticket.ParkingLot,
		ParkedDurationMinutes: &minutes,
		Charge:                &charge,
	}

	// Remove the ticket from storage
	h.service.RemoveTicket(c, params.TicketId)

	c.JSON(http.StatusOK, response)
}
