package ocpp

import (
	"context"
	"fmt"
	"time"

	"github.com/balu-dk/go-cpms/config"
	"github.com/balu-dk/go-cpms/internal/db"
	"github.com/balu-dk/go-cpms/internal/db/models"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/sirupsen/logrus"
)

// CentralSystem manages the OCPP central system
type CentralSystem struct {
	OcppServer ocpp16.CentralSystem
	db         *db.PostgresStore
	logger     *OCPPLogger
	config     *config.Config
}

// NewCentralSystem creates a new OCPP central system
func NewCentralSystem(cfg *config.Config, store *db.PostgresStore) *CentralSystem {
	cs := &CentralSystem{
		OcppServer: ocpp16.NewCentralSystem(nil, nil),
		db:         store,
		logger:     NewOCPPLogger(store),
		config:     cfg,
	}

	// Set up OCPP handlers
	centralSystemHandler := &CentralSystemHandler{
		cs: cs,
	}
	cs.OcppServer.SetCoreHandler(centralSystemHandler)
	cs.OcppServer.SetFirmwareManagementHandler(centralSystemHandler)

	// Set up connection handlers
	cs.OcppServer.SetNewChargePointHandler(cs.handleNewChargePoint)
	cs.OcppServer.SetChargePointDisconnectedHandler(cs.handleChargePointDisconnected)

	return cs
}

// Start starts the OCPP central system
func (cs *CentralSystem) Start() error {
	logrus.Infof("Starting OCPP central system on port %d with path %s", cs.config.ServerPort, cs.config.OCPPPath)
	cs.OcppServer.Start(cs.config.ServerPort, cs.config.OCPPPath)
	return nil
}

// handleNewChargePoint handles a new charge point connection
func (cs *CentralSystem) handleNewChargePoint(cp ocpp16.ChargePointConnection) {
	logrus.WithField("chargePointID", cp.ID()).Info("New charge point connected")

	// Create a new charge point record or update the existing one
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get existing charge point or create a minimal record
	// Full details will be updated when BootNotification is received
	chargePoint, err := cs.db.GetChargePoint(ctx, cp.ID())
	if err != nil {
		// Create a minimal new charge point record
		chargePoint = &models.ChargePoint{
			ID:                 cp.ID(),
			Vendor:             "Unknown",
			Model:              "Unknown",
			RegistrationStatus: "Pending",
			IsConnected:        true,
			ConnectedSince:     time.Now(),
		}
	} else {
		// Update connection status
		chargePoint.IsConnected = true
		chargePoint.ConnectedSince = time.Now()
	}

	if err := cs.db.SaveChargePoint(ctx, chargePoint); err != nil {
		logrus.WithError(err).WithField("chargePointID", cp.ID()).Error("Failed to save charge point")
	}
}

// handleChargePointDisconnected handles a charge point disconnection
func (cs *CentralSystem) handleChargePointDisconnected(cp ocpp16.ChargePointConnection) {
	logrus.WithField("chargePointID", cp.ID()).Info("Charge point disconnected")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.db.UpdateChargePointConnection(ctx, cp.ID(), false); err != nil {
		logrus.WithError(err).WithField("chargePointID", cp.ID()).Error("Failed to update charge point connection status")
	}
}

// CentralSystemHandler implements the OCPP handlers
type CentralSystemHandler struct {
	cs *CentralSystem
}

// OnBootNotification handles BootNotification requests
func (h *CentralSystemHandler) OnBootNotification(chargePointID string, request *core.BootNotificationRequest) (confirmation *core.BootNotificationConfirmation, err error) {
	logrus.WithFields(logrus.Fields{
		"chargePointID": chargePointID,
		"vendor":        request.ChargePointVendor,
		"model":         request.ChargePointModel,
	}).Info("Boot notification received")

	// Log the request
	h.cs.logger.LogRequest(chargePointID, "BootNotification", "", request, "Inbound")

	// Update charge point in database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	chargePoint := &models.ChargePoint{
		ID:                 chargePointID,
		Vendor:             request.ChargePointVendor,
		Model:              request.ChargePointModel,
		SerialNumber:       request.ChargePointSerialNumber,
		FirmwareVersion:    request.FirmwareVersion,
		LastHeartbeat:      time.Now(),
		RegistrationStatus: string(core.RegistrationStatusAccepted),
		IsConnected:        true,
		ConnectedSince:     time.Now(),
	}

	if err := h.cs.db.SaveChargePoint(ctx, chargePoint); err != nil {
		logrus.WithError(err).WithField("chargePointID", chargePointID).Error("Failed to save charge point")
	}

	// Create response
	conf := core.NewBootNotificationConfirmation(
		types.NewDateTime(time.Now()),
		h.cs.config.HeartbeatInterval,
		core.RegistrationStatusAccepted,
	)

	// Log the response
	h.cs.logger.LogResponse(chargePointID, "BootNotification", "", conf, "Outbound")

	return conf, nil
}

// OnHeartbeat handles Heartbeat requests
func (h *CentralSystemHandler) OnHeartbeat(chargePointID string, request *core.HeartbeatRequest) (confirmation *core.HeartbeatConfirmation, err error) {
	logrus.WithField("chargePointID", chargePointID).Debug("Heartbeat received")

	// Log the request
	h.cs.logger.LogRequest(chargePointID, "Heartbeat", "", request, "Inbound")

	// Update last heartbeat time
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.cs.db.UpdateHeartbeat(ctx, chargePointID); err != nil {
		logrus.WithError(err).WithField("chargePointID", chargePointID).Error("Failed to update heartbeat")
	}

	// Create response
	conf := core.NewHeartbeatConfirmation(types.NewDateTime(time.Now()))

	// Log the response
	h.cs.logger.LogResponse(chargePointID, "Heartbeat", "", conf, "Outbound")

	return conf, nil
}

// OnStatusNotification handles StatusNotification requests
func (h *CentralSystemHandler) OnStatusNotification(chargePointID string, request *core.StatusNotificationRequest) (confirmation *core.StatusNotificationConfirmation, err error) {
	logrus.WithFields(logrus.Fields{
		"chargePointID": chargePointID,
		"connectorId":   request.ConnectorId,
		"status":        request.Status,
		"errorCode":     request.ErrorCode,
	}).Info("Status notification received")

	// Log the request
	h.cs.logger.LogRequest(chargePointID, "StatusNotification", "", request, "Inbound")

	// Update connector status in database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connector := &models.Connector{
		ID:            request.ConnectorId,
		ChargePointID: chargePointID,
		Status:        string(request.Status),
		ErrorCode:     string(request.ErrorCode),
	}

	if err := h.cs.db.SaveConnector(ctx, connector); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"connectorId":   request.ConnectorId,
		}).Error("Failed to save connector status")
	}

	// Create response
	conf := core.NewStatusNotificationConfirmation()

	// Log the response
	h.cs.logger.LogResponse(chargePointID, "StatusNotification", "", conf, "Outbound")

	return conf, nil
}

// OnMeterValues handles MeterValues requests
func (h *CentralSystemHandler) OnMeterValues(chargePointID string, request *core.MeterValuesRequest) (confirmation *core.MeterValuesConfirmation, err error) {
	logrus.WithFields(logrus.Fields{
		"chargePointID": chargePointID,
		"connectorId":   request.ConnectorId,
	}).Debug("Meter values received")

	// Log the request
	h.cs.logger.LogRequest(chargePointID, "MeterValues", "", request, "Inbound")

	// Process meter values
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, meterValue := range request.MeterValue {
		for _, sampledValue := range meterValue.SampledValue {
			// Handle only power consumption values by default
			measurand := "Energy.Active.Import.Register"
			if sampledValue.Measurand != "" {
				measurand = string(sampledValue.Measurand)
			}

			unit := "Wh"
			if sampledValue.Unit != "" {
				unit = string(sampledValue.Unit)
			}

			value := 0.0
			if v, err := parseFloat64(sampledValue.Value); err == nil {
				value = v
			}

			mv := &models.MeterValue{
				ChargePointID: chargePointID,
				ConnectorID:   request.ConnectorId,
				Timestamp:     meterValue.Timestamp.Time,
				Value:         value,
				Unit:          unit,
				Measurand:     measurand,
			}

			if request.TransactionId != nil {
				mv.TransactionID = *request.TransactionId
			}

			if err := h.cs.db.SaveMeterValue(ctx, mv); err != nil {
				logrus.WithError(err).WithFields(logrus.Fields{
					"chargePointID": chargePointID,
					"connectorId":   request.ConnectorId,
				}).Error("Failed to save meter value")
			}
		}
	}

	// Create response
	conf := core.NewMeterValuesConfirmation()

	// Log the response
	h.cs.logger.LogResponse(chargePointID, "MeterValues", "", conf, "Outbound")

	return conf, nil
}

// OnStartTransaction handles StartTransaction requests
func (h *CentralSystemHandler) OnStartTransaction(chargePointID string, request *core.StartTransactionRequest) (confirmation *core.StartTransactionConfirmation, err error) {
	logrus.WithFields(logrus.Fields{
		"chargePointID": chargePointID,
		"connectorId":   request.ConnectorId,
		"idTag":         request.IdTag,
	}).Info("Start transaction request received")

	// Log the request
	h.cs.logger.LogRequest(chargePointID, "StartTransaction", "", request, "Inbound")

	// Save transaction in database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transaction := &models.Transaction{
		ID:            generateTransactionID(),
		ChargePointID: chargePointID,
		ConnectorID:   request.ConnectorId,
		IdTag:         request.IdTag,
		StartTime:     request.Timestamp.Time,
		MeterStart:    request.MeterStart,
		Status:        "InProgress",
	}

	if err := h.cs.db.StartTransaction(ctx, transaction); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"connectorId":   request.ConnectorId,
		}).Error("Failed to save transaction")
	}

	// Create response
	idTagInfo := types.NewIdTagInfo(types.AuthorizationStatusAccepted)
	conf := core.NewStartTransactionConfirmation(idTagInfo, transaction.ID)

	// Log the response
	h.cs.logger.LogResponse(chargePointID, "StartTransaction", "", conf, "Outbound")

	return conf, nil
}

// OnStopTransaction handles StopTransaction requests
func (h *CentralSystemHandler) OnStopTransaction(chargePointID string, request *core.StopTransactionRequest) (confirmation *core.StopTransactionConfirmation, err error) {
	logrus.WithFields(logrus.Fields{
		"chargePointID": chargePointID,
		"transactionId": request.TransactionId,
	}).Info("Stop transaction request received")

	// Log the request
	h.cs.logger.LogRequest(chargePointID, "StopTransaction", "", request, "Inbound")

	// Update transaction in database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.cs.db.StopTransaction(ctx, request.TransactionId, request.Timestamp.Time, request.MeterStop); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"transactionId": request.TransactionId,
		}).Error("Failed to update transaction")
	}

	// Process any transaction-specific meter values
	if request.TransactionData != nil {
		for _, meterValue := range request.TransactionData {
			for _, sampledValue := range meterValue.SampledValue {
				measurand := "Energy.Active.Import.Register"
				if sampledValue.Measurand != "" {
					measurand = string(sampledValue.Measurand)
				}

				unit := "Wh"
				if sampledValue.Unit != "" {
					unit = string(sampledValue.Unit)
				}

				value := 0.0
				if v, err := parseFloat64(sampledValue.Value); err == nil {
					value = v
				}

				mv := &models.MeterValue{
					TransactionID: request.TransactionId,
					ChargePointID: chargePointID,
					ConnectorID:   0, // We don't have connector ID in stop transaction
					Timestamp:     meterValue.Timestamp.Time,
					Value:         value,
					Unit:          unit,
					Measurand:     measurand,
				}

				if err := h.cs.db.SaveMeterValue(ctx, mv); err != nil {
					logrus.WithError(err).WithFields(logrus.Fields{
						"chargePointID": chargePointID,
						"transactionId": request.TransactionId,
					}).Error("Failed to save transaction meter value")
				}
			}
		}
	}

	// Create response
	conf := core.NewStopTransactionConfirmation()

	// Log the response
	h.cs.logger.LogResponse(chargePointID, "StopTransaction", "", conf, "Outbound")

	return conf, nil
}

// OnAuthorize handles Authorize requests
func (h *CentralSystemHandler) OnAuthorize(chargePointID string, request *core.AuthorizeRequest) (confirmation *core.AuthorizeConfirmation, err error) {
	logrus.WithFields(logrus.Fields{
		"chargePointID": chargePointID,
		"idTag":         request.IdTag,
	}).Info("Authorize request received")

	// Log the request
	h.cs.logger.LogRequest(chargePointID, "Authorize", "", request, "Inbound")

	// In a real system, we would check if the ID tag is authorized
	// For simplicity, we accept all authorize requests
	idTagInfo := types.NewIdTagInfo(types.AuthorizationStatusAccepted)
	conf := core.NewAuthorizationConfirmation(idTagInfo)

	// Log the response
	h.cs.logger.LogResponse(chargePointID, "Authorize", "", conf, "Outbound")

	return conf, nil
}

// OnDataTransfer handles DataTransfer requests
func (h *CentralSystemHandler) OnDataTransfer(chargePointID string, request *core.DataTransferRequest) (confirmation *core.DataTransferConfirmation, err error) {
	logrus.WithFields(logrus.Fields{
		"chargePointID": chargePointID,
		"vendorId":      request.VendorId,
		"messageId":     request.MessageId,
	}).Info("Data transfer request received")

	// Log the request
	h.cs.logger.LogRequest(chargePointID, "DataTransfer", "", request, "Inbound")

	// For simplicity, we accept all data transfer requests
	conf := core.NewDataTransferConfirmation(core.DataTransferStatusAccepted)

	// Log the response
	h.cs.logger.LogResponse(chargePointID, "DataTransfer", "", conf, "Outbound")

	return conf, nil
}

// OnDiagnosticsStatusNotification handles DiagnosticsStatusNotification requests
func (h *CentralSystemHandler) OnDiagnosticsStatusNotification(chargePointID string, request *firmware.DiagnosticsStatusNotificationRequest) (confirmation *firmware.DiagnosticsStatusNotificationConfirmation, err error) {
	logrus.WithFields(logrus.Fields{
		"chargePointID": chargePointID,
		"status":        request.Status,
	}).Info("Diagnostics status notification received")

	// Log the request
	h.cs.logger.LogRequest(chargePointID, "DiagnosticsStatusNotification", "", request, "Inbound")

	// Create response
	conf := firmware.NewDiagnosticsStatusNotificationConfirmation()

	// Log the response
	h.cs.logger.LogResponse(chargePointID, "DiagnosticsStatusNotification", "", conf, "Outbound")

	return conf, nil
}

// OnFirmwareStatusNotification handles FirmwareStatusNotification requests
func (h *CentralSystemHandler) OnFirmwareStatusNotification(chargePointID string, request *firmware.FirmwareStatusNotificationRequest) (confirmation *firmware.FirmwareStatusNotificationConfirmation, err error) {
	logrus.WithFields(logrus.Fields{
		"chargePointID": chargePointID,
		"status":        request.Status,
	}).Info("Firmware status notification received")

	// Log the request
	h.cs.logger.LogRequest(chargePointID, "FirmwareStatusNotification", "", request, "Inbound")

	// Create response
	conf := firmware.NewFirmwareStatusNotificationConfirmation()

	// Log the response
	h.cs.logger.LogResponse(chargePointID, "FirmwareStatusNotification", "", conf, "Outbound")

	return conf, nil
}

// Helper function to generate a unique transaction ID
// In a production system, you would use a more robust method
var lastTransactionID = 1000

func generateTransactionID() int {
	lastTransactionID++
	return lastTransactionID
}

// Helper function to parse a string to float64
func parseFloat64(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
