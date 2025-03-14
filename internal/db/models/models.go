package models

import (
	"time"
)

// ChargePoint represents a charge point connected to the CPMS
type ChargePoint struct {
	ID                 string    `json:"id"`
	Vendor             string    `json:"vendor"`
	Model              string    `json:"model"`
	SerialNumber       string    `json:"serialNumber"`
	FirmwareVersion    string    `json:"firmwareVersion"`
	LastHeartbeat      time.Time `json:"lastHeartbeat"`
	RegistrationStatus string    `json:"registrationStatus"`
	ConnectedSince     time.Time `json:"connectedSince"`
	IsConnected        bool      `json:"isConnected"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// Connector represents a connector/plug on a charge point
type Connector struct {
	ID            int       `json:"id"`
	ChargePointID string    `json:"chargePointId"`
	Status        string    `json:"status"`
	ErrorCode     string    `json:"errorCode"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Transaction represents a charging transaction
type Transaction struct {
	ID            int       `json:"id"`
	ChargePointID string    `json:"chargePointId"`
	ConnectorID   int       `json:"connectorId"`
	IdTag         string    `json:"idTag"`
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime,omitempty"`
	MeterStart    int       `json:"meterStart"`
	MeterStop     int       `json:"meterStop,omitempty"`
	Status        string    `json:"status"` // InProgress, Completed, Stopped
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// OCPPMessage represents a logged OCPP message
type OCPPMessage struct {
	ID            int       `json:"id"`
	ChargePointID string    `json:"chargePointId"`
	MessageType   string    `json:"messageType"` // Request or Response
	Action        string    `json:"action"`      // OCPP action like BootNotification, StatusNotification, etc.
	RequestID     string    `json:"requestId"`
	Payload       string    `json:"payload"`   // JSON string of the message
	Direction     string    `json:"direction"` // Inbound or Outbound
	Timestamp     time.Time `json:"timestamp"`
}

// MeterValue represents meter readings from a charge point
type MeterValue struct {
	ID            int       `json:"id"`
	TransactionID int       `json:"transactionId"`
	ChargePointID string    `json:"chargePointId"`
	ConnectorID   int       `json:"connectorId"`
	Timestamp     time.Time `json:"timestamp"`
	Value         float64   `json:"value"`
	Unit          string    `json:"unit"`
	Measurand     string    `json:"measurand"`
	CreatedAt     time.Time `json:"createdAt"`
}
