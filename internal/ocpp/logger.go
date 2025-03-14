package ocpp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/balu-dk/go-cpms/internal/db"
	"github.com/balu-dk/go-cpms/internal/db/models"
	"github.com/sirupsen/logrus"
)

// OCPPLogger logs OCPP messages to the database
type OCPPLogger struct {
	db *db.PostgresStore
}

// NewOCPPLogger creates a new OCPP logger
func NewOCPPLogger(db *db.PostgresStore) *OCPPLogger {
	return &OCPPLogger{
		db: db,
	}
}

// LogRequest logs an OCPP request
func (l *OCPPLogger) LogRequest(chargePointID, action, requestID string, payload interface{}, direction string) {
	l.logMessage(chargePointID, "Request", action, requestID, payload, direction)
}

// LogResponse logs an OCPP response
func (l *OCPPLogger) LogResponse(chargePointID, action, requestID string, payload interface{}, direction string) {
	l.logMessage(chargePointID, "Response", action, requestID, payload, direction)
}

// logMessage logs an OCPP message to the database
func (l *OCPPLogger) logMessage(chargePointID, messageType, action, requestID string, payload interface{}, direction string) {
	// Konverter payload til en JSON-string
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal OCPP message payload")
		payloadJSON = []byte("{}")
	}

	msg := &models.OCPPMessage{
		ChargePointID: chargePointID,
		MessageType:   messageType,
		Action:        action,
		RequestID:     requestID,
		Payload:       string(payloadJSON), // Konverteret til string
		Direction:     direction,
		Timestamp:     time.Now(),
	}

	// Use a background context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := l.db.LogOCPPMessage(ctx, msg); err != nil {
		logrus.WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"action":        action,
			"requestID":     requestID,
			"error":         err,
		}).Error("Failed to log OCPP message")
	}
}
