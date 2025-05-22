package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"parking-lot/internal/logger"
	"parking-lot/internal/model"
	"parking-lot/internal/service"
	"parking-lot/server/api"
)

// ParkingHandler implements the ServerInterface
type ParkingHandler struct {
	service service.ParkingLotServicer
	log     logger.Logger
}

// NewParkingHandler creates a new handler with the given service
func NewParkingHandler(service service.ParkingLotServicer) *ParkingHandler {
	return &ParkingHandler{
		service: service,
		log:     logger.NewLogger(),
	}
}

// PostEntry records a vehicle entry and generates a ticket
func (h *ParkingHandler) PostEntry(c *gin.Context, params api.PostEntryParams) {
	ctx := c.Request.Context()

	log := h.log.WithContext(ctx).WithFields(
		logger.Field{Key: "plate", Value: params.Plate},
		logger.Field{Key: "parking_lot", Value: params.ParkingLot},
	)
	log.Info("Processing vehicle entry")

	ticketID, _ := h.service.CreateTicket(ctx, params.Plate, params.ParkingLot)

	// Return the ticket ID
	response := api.EntryResponse{
		TicketId: ticketID,
	}

	log.Info("Vehicle entry processed successfully",
		logger.Field{Key: "ticket_id", Value: ticketID.String()},
	)
	c.JSON(http.StatusOK, response)
}

// PostExit processes a vehicle exit
func (h *ParkingHandler) PostExit(c *gin.Context, params api.PostExitParams) {
	ctx := c.Request.Context()

	log := h.log.WithContext(ctx).WithFields(
		logger.Field{Key: "ticket_id", Value: params.TicketId},
	)
	log.Info("Processing vehicle exit")

	ticket, exists := h.service.GetTicket(ctx, params.TicketId.String())
	if !exists {
		errorMsg := "Ticket not found"
		response := api.ErrorResponse{
			Message: errorMsg,
		}
		log.Warn("Ticket not found")
		c.JSON(http.StatusNotFound, response)
		return
	}

	// Calculate parking duration and charge
	minutes, charge := h.service.CalculateCharge(ticket.EntryTime)

	log.Info("Calculated parking charge",
		logger.Field{Key: "minutes", Value: minutes},
		logger.Field{Key: "charge", Value: charge},
	)

	// Update ticket status and charge
	ticket.Status = model.TicketStatusOut // Assuming model.TicketStatusOut is defined
	ticket.Charge = charge

	// Update the ticket in storage
	if err := h.service.UpdateTicket(ctx, ticket); err != nil {
		errorMsg := "Failed to update ticket"
		response := api.ErrorResponse{
			Message: errorMsg,
		}
		log.Error("Failed to update ticket", logger.Field{Key: "error", Value: err.Error()})
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// Create response
	response := api.ExitResponse{
		Plate:                 ticket.Plate,
		ParkingLot:            ticket.ParkingLot,
		ParkedDurationMinutes: minutes,
		Charge:                charge,
	}

	log.Info("Vehicle exit processed successfully")
	c.JSON(http.StatusOK, response)
}
