package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/balu-dk/go-cpms/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

// Handler handles API requests
type Handler struct {
	cpms *service.CPMS
}

// NewHandler creates a new API handler
func NewHandler(cpms *service.CPMS) *Handler {
	return &Handler{
		cpms: cpms,
	}
}

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// GetChargePoints returns all charge points
func (h *Handler) GetChargePoints(w http.ResponseWriter, r *http.Request) {
	chargePoints, err := h.cpms.GetChargePoints(r.Context())
	if err != nil {
		logrus.WithError(err).Error("Failed to get charge points")
		sendErrorResponse(w, "Failed to get charge points", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Data:    chargePoints,
	})
}

// GetChargePoint returns a specific charge point
func (h *Handler) GetChargePoint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendErrorResponse(w, "Charge point ID is required", http.StatusBadRequest)
		return
	}

	chargePoint, err := h.cpms.GetChargePoint(r.Context(), id)
	if err != nil {
		logrus.WithError(err).WithField("id", id).Error("Failed to get charge point")
		sendErrorResponse(w, "Failed to get charge point", http.StatusInternalServerError)
		return
	}

	if chargePoint == nil {
		sendErrorResponse(w, "Charge point not found", http.StatusNotFound)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Data:    chargePoint,
	})
}

// GetConnectors returns all connectors for a charge point
func (h *Handler) GetConnectors(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendErrorResponse(w, "Charge point ID is required", http.StatusBadRequest)
		return
	}

	connectors, err := h.cpms.GetConnectors(r.Context(), id)
	if err != nil {
		logrus.WithError(err).WithField("id", id).Error("Failed to get connectors")
		sendErrorResponse(w, "Failed to get connectors", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Data:    connectors,
	})
}

// Reset resets a charge point
func (h *Handler) Reset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendErrorResponse(w, "Charge point ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Type string `json:"type"` // "Hard" or "Soft"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Type != "Hard" && req.Type != "Soft" {
		sendErrorResponse(w, "Type must be 'Hard' or 'Soft'", http.StatusBadRequest)
		return
	}

	if err := h.cpms.ResetChargePoint(r.Context(), id, req.Type); err != nil {
		logrus.WithError(err).WithField("id", id).Error("Failed to reset charge point")
		sendErrorResponse(w, "Failed to reset charge point", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Message: "Reset command sent",
	})
}

// ChangeAvailability changes the availability of a connector
func (h *Handler) ChangeAvailability(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendErrorResponse(w, "Charge point ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		ConnectorID int    `json:"connectorId"`
		Type        string `json:"type"` // "Operative" or "Inoperative"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ConnectorID < 0 {
		sendErrorResponse(w, "ConnectorID must be non-negative", http.StatusBadRequest)
		return
	}

	if req.Type != "Operative" && req.Type != "Inoperative" {
		sendErrorResponse(w, "Type must be 'Operative' or 'Inoperative'", http.StatusBadRequest)
		return
	}

	if err := h.cpms.ChangeAvailability(r.Context(), id, req.ConnectorID, req.Type); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"id":          id,
			"connectorID": req.ConnectorID,
		}).Error("Failed to change availability")
		sendErrorResponse(w, "Failed to change availability", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Message: "Change availability command sent",
	})
}

// UnlockConnector unlocks a connector
func (h *Handler) UnlockConnector(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendErrorResponse(w, "Charge point ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		ConnectorID int `json:"connectorId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ConnectorID <= 0 {
		sendErrorResponse(w, "ConnectorID must be positive", http.StatusBadRequest)
		return
	}

	if err := h.cpms.UnlockConnector(r.Context(), id, req.ConnectorID); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"id":          id,
			"connectorID": req.ConnectorID,
		}).Error("Failed to unlock connector")
		sendErrorResponse(w, "Failed to unlock connector", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Message: "Unlock connector command sent",
	})
}

// RemoteStartTransaction starts a transaction remotely
func (h *Handler) RemoteStartTransaction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendErrorResponse(w, "Charge point ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		ConnectorID int    `json:"connectorId"`
		IdTag       string `json:"idTag"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ConnectorID <= 0 {
		sendErrorResponse(w, "ConnectorID must be positive", http.StatusBadRequest)
		return
	}

	if req.IdTag == "" {
		sendErrorResponse(w, "IdTag is required", http.StatusBadRequest)
		return
	}

	if err := h.cpms.RemoteStartTransaction(r.Context(), id, req.ConnectorID, req.IdTag); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"id":          id,
			"connectorID": req.ConnectorID,
			"idTag":       req.IdTag,
		}).Error("Failed to start transaction")
		sendErrorResponse(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Message: "Remote start transaction command sent",
	})
}

// RemoteStopTransaction stops a transaction remotely
func (h *Handler) RemoteStopTransaction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendErrorResponse(w, "Charge point ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		TransactionID int `json:"transactionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TransactionID <= 0 {
		sendErrorResponse(w, "TransactionID must be positive", http.StatusBadRequest)
		return
	}

	if err := h.cpms.RemoteStopTransaction(r.Context(), id, req.TransactionID); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"id":            id,
			"transactionID": req.TransactionID,
		}).Error("Failed to stop transaction")
		sendErrorResponse(w, "Failed to stop transaction", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Message: "Remote stop transaction command sent",
	})
}

// TriggerHeartbeat triggers a heartbeat from a charge point
func (h *Handler) TriggerHeartbeat(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendErrorResponse(w, "Charge point ID is required", http.StatusBadRequest)
		return
	}

	if err := h.cpms.TriggerHeartbeat(r.Context(), id); err != nil {
		logrus.WithError(err).WithField("id", id).Error("Failed to trigger heartbeat")
		sendErrorResponse(w, "Failed to trigger heartbeat", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Message: "Trigger heartbeat command sent",
	})
}

// GetTransaction gets a transaction
func (h *Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		sendErrorResponse(w, "Transaction ID is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		sendErrorResponse(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}

	transaction, err := h.cpms.GetTransaction(r.Context(), id)
	if err != nil {
		logrus.WithError(err).WithField("id", id).Error("Failed to get transaction")
		sendErrorResponse(w, "Failed to get transaction", http.StatusInternalServerError)
		return
	}

	if transaction == nil {
		sendErrorResponse(w, "Transaction not found", http.StatusNotFound)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Data:    transaction,
	})
}

// GetDiagnostics requests the charge point to upload diagnostics
func (h *Handler) GetDiagnostics(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendErrorResponse(w, "Charge point ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Location  string `json:"location"`
		StartTime string `json:"startTime,omitempty"`
		StopTime  string `json:"stopTime,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Location == "" {
		sendErrorResponse(w, "Location is required", http.StatusBadRequest)
		return
	}

	var startTime, stopTime time.Time
	var err error

	if req.StartTime != "" {
		startTime, err = time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			sendErrorResponse(w, "Invalid startTime format, use RFC3339", http.StatusBadRequest)
			return
		}
	}

	if req.StopTime != "" {
		stopTime, err = time.Parse(time.RFC3339, req.StopTime)
		if err != nil {
			sendErrorResponse(w, "Invalid stopTime format, use RFC3339", http.StatusBadRequest)
			return
		}
	}

	if err := h.cpms.GetDiagnostics(r.Context(), id, req.Location, startTime, stopTime); err != nil {
		logrus.WithError(err).WithField("id", id).Error("Failed to get diagnostics")
		sendErrorResponse(w, "Failed to get diagnostics", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Message: "Get diagnostics command sent",
	})
}

// UpdateFirmware requests the charge point to update its firmware
func (h *Handler) UpdateFirmware(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendErrorResponse(w, "Charge point ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Location     string `json:"location"`
		RetrieveDate string `json:"retrieveDate"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Location == "" {
		sendErrorResponse(w, "Location is required", http.StatusBadRequest)
		return
	}

	if req.RetrieveDate == "" {
		sendErrorResponse(w, "RetrieveDate is required", http.StatusBadRequest)
		return
	}

	retrieveDate, err := time.Parse(time.RFC3339, req.RetrieveDate)
	if err != nil {
		sendErrorResponse(w, "Invalid retrieveDate format, use RFC3339", http.StatusBadRequest)
		return
	}

	if err := h.cpms.UpdateFirmware(r.Context(), id, req.Location, retrieveDate); err != nil {
		logrus.WithError(err).WithField("id", id).Error("Failed to update firmware")
		sendErrorResponse(w, "Failed to update firmware", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Message: "Update firmware command sent",
	})
}

// ClearCache requests the charge point to clear its cache
func (h *Handler) ClearCache(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendErrorResponse(w, "Charge point ID is required", http.StatusBadRequest)
		return
	}

	if err := h.cpms.ClearCache(r.Context(), id); err != nil {
		logrus.WithError(err).WithField("id", id).Error("Failed to clear cache")
		sendErrorResponse(w, "Failed to clear cache", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Message: "Clear cache command sent",
	})
}

// GetConfiguration gets the charge point's configuration
func (h *Handler) GetConfiguration(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendErrorResponse(w, "Charge point ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Keys []string `json:"keys,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.cpms.GetConfiguration(r.Context(), id, req.Keys); err != nil {
		logrus.WithError(err).WithField("id", id).Error("Failed to get configuration")
		sendErrorResponse(w, "Failed to get configuration", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Message: "Get configuration command sent",
	})
}

// ChangeConfiguration changes a configuration key on the charge point
func (h *Handler) ChangeConfiguration(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendErrorResponse(w, "Charge point ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Key == "" {
		sendErrorResponse(w, "Key is required", http.StatusBadRequest)
		return
	}

	if err := h.cpms.ChangeConfiguration(r.Context(), id, req.Key, req.Value); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"id":  id,
			"key": req.Key,
		}).Error("Failed to change configuration")
		sendErrorResponse(w, "Failed to change configuration", http.StatusInternalServerError)
		return
	}

	sendResponse(w, Response{
		Success: true,
		Message: "Change configuration command sent",
	})
}

// Helper functions to send responses
func sendResponse(w http.ResponseWriter, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.WithError(err).Error("Failed to encode response")
	}
}

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(ErrorResponse{
		Success: false,
		Error:   message,
	}); err != nil {
		logrus.WithError(err).Error("Failed to encode error response")
	}
}
