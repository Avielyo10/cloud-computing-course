package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"parking-lot/server/api"
)

// dummyServer implements api.ServerInterface for testing
type dummyServer struct {
	lastEntryParams api.PostEntryParams
	lastExitParams  api.PostExitParams
}

func (d *dummyServer) PostEntry(c *gin.Context, params api.PostEntryParams) {
	d.lastEntryParams = params
	c.JSON(http.StatusOK, gin.H{
		"plate":      params.Plate,
		"parkingLot": params.ParkingLot,
	})
}

func (d *dummyServer) PostExit(c *gin.Context, params api.PostExitParams) {
	d.lastExitParams = params
	c.JSON(http.StatusOK, gin.H{
		"ticketId": params.TicketId.String(),
	})
}

func setupRouter(si api.ServerInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	a := si
	api.RegisterHandlers(r, a)
	return r
}

func TestPostEntry_MissingPlate(t *testing.T) {
	r := setupRouter(&dummyServer{})
	req := httptest.NewRequest("POST", "/entry", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `plate is required`)
}

func TestPostEntry_InvalidParkingLot(t *testing.T) {
	r := setupRouter(&dummyServer{})
	req := httptest.NewRequest("POST", "/entry?plate=foo&parkingLot=notint", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `Invalid format for parameter parkingLot`)
}

func TestPostEntry_Success(t *testing.T) {
	d := &dummyServer{}
	r := setupRouter(d)
	// Use query parameters
	url := "/entry?plate=bar&parkingLot=123"
	req := httptest.NewRequest("POST", url, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Response JSON should have plate and parkingLot
	assert.Contains(t, w.Body.String(), `"plate":"bar"`)
	assert.Contains(t, w.Body.String(), `"parkingLot":123`)
	// dummyServer should have lastEntryParams set
	assert.Equal(t, "bar", d.lastEntryParams.Plate)
	assert.Equal(t, 123, d.lastEntryParams.ParkingLot)
}

func TestPostExit_MissingTicketID(t *testing.T) {
	r := setupRouter(&dummyServer{})
	req := httptest.NewRequest("POST", "/exit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `ticketId is required`)
}

func TestPostExit_InvalidTicketID(t *testing.T) {
	r := setupRouter(&dummyServer{})
	req := httptest.NewRequest("POST", "/exit?ticketId=notuuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `Invalid format for parameter ticketId`)
}

func TestPostExit_Success(t *testing.T) {
	d := &dummyServer{}
	r := setupRouter(d)
	// Create a valid UUID
	req := httptest.NewRequest("POST", "/exit?ticketId=00000000-0000-0000-0000-000000000000", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Response JSON should contain ticketId
	assert.Contains(t, w.Body.String(), `"ticketId":"00000000-0000-0000-0000-000000000000"`)
}
