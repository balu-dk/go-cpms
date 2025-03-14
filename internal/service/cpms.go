package service

import (
	"context"
	"fmt"
	"time"

	"github.com/balu-dk/go-cpms/config"
	"github.com/balu-dk/go-cpms/internal/db"
	"github.com/balu-dk/go-cpms/internal/db/models"
	"github.com/balu-dk/go-cpms/internal/ocpp"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/sirupsen/logrus"
)

// CPMS represents the Charging Point Management System service
type CPMS struct {
	config        *config.Config
	db            *db.PostgresStore
	centralSystem *ocpp.CentralSystem
}

// NewCPMS creates a new CPMS service
func NewCPMS(cfg *config.Config, store *db.PostgresStore) *CPMS {
	return &CPMS{
		config: cfg,
		db:     store,
	}
}

// Start starts the CPMS service
func (s *CPMS) Start() error {
	// Start the central system
	s.centralSystem = ocpp.NewCentralSystem(s.config, s.db)
	return s.centralSystem.Start()
}

// GetChargePoints returns all charge points
func (s *CPMS) GetChargePoints(ctx context.Context) ([]*models.ChargePoint, error) {
	return s.db.GetAllChargePoints(ctx)
}

// GetChargePoint returns a specific charge point
func (s *CPMS) GetChargePoint(ctx context.Context, id string) (*models.ChargePoint, error) {
	return s.db.GetChargePoint(ctx, id)
}

// GetConnectors returns all connectors for a charge point
func (s *CPMS) GetConnectors(ctx context.Context, chargePointID string) ([]*models.Connector, error) {
	return s.db.GetConnectors(ctx, chargePointID)
}

// GetTransaction returns a specific transaction
func (s *CPMS) GetTransaction(ctx context.Context, id int) (*models.Transaction, error) {
	return s.db.GetTransaction(ctx, id)
}

// ResetChargePoint sends a reset request to a charge point
func (s *CPMS) ResetChargePoint(ctx context.Context, chargePointID string, resetType string) error {
	var ocppResetType core.ResetType
	switch resetType {
	case "Hard":
		ocppResetType = core.ResetTypeHard
	case "Soft":
		ocppResetType = core.ResetTypeSoft
	default:
		return fmt.Errorf("invalid reset type: %s", resetType)
	}

	callback := func(confirmation *core.ResetConfirmation, err error) {
		if err != nil {
			logrus.WithError(err).WithField("chargePointID", chargePointID).Error("Reset request failed")
			return
		}

		logrus.WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"status":        confirmation.Status,
		}).Info("Reset request processed")
	}

	return s.centralSystem.OcppServer.Reset(chargePointID, callback, ocppResetType)
}

// ChangeAvailability changes the availability of a connector
func (s *CPMS) ChangeAvailability(ctx context.Context, chargePointID string, connectorID int, availabilityType string) error {
	var ocppAvailabilityType core.AvailabilityType
	switch availabilityType {
	case "Operative":
		ocppAvailabilityType = core.AvailabilityTypeOperative
	case "Inoperative":
		ocppAvailabilityType = core.AvailabilityTypeInoperative
	default:
		return fmt.Errorf("invalid availability type: %s", availabilityType)
	}

	callback := func(confirmation *core.ChangeAvailabilityConfirmation, err error) {
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"chargePointID": chargePointID,
				"connectorID":   connectorID,
			}).Error("Change availability request failed")
			return
		}

		logrus.WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"connectorID":   connectorID,
			"status":        confirmation.Status,
		}).Info("Change availability request processed")
	}

	return s.centralSystem.OcppServer.ChangeAvailability(chargePointID, callback, connectorID, ocppAvailabilityType)
}

// UnlockConnector sends an unlock connector request
func (s *CPMS) UnlockConnector(ctx context.Context, chargePointID string, connectorID int) error {
	callback := func(confirmation *core.UnlockConnectorConfirmation, err error) {
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"chargePointID": chargePointID,
				"connectorID":   connectorID,
			}).Error("Unlock connector request failed")
			return
		}

		logrus.WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"connectorID":   connectorID,
			"status":        confirmation.Status,
		}).Info("Unlock connector request processed")
	}

	return s.centralSystem.OcppServer.UnlockConnector(chargePointID, callback, connectorID)
}

// RemoteStartTransaction sends a remote start transaction request
func (s *CPMS) RemoteStartTransaction(ctx context.Context, chargePointID string, connectorID int, idTag string) error {
	callback := func(confirmation *core.RemoteStartTransactionConfirmation, err error) {
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"chargePointID": chargePointID,
				"connectorID":   connectorID,
				"idTag":         idTag,
			}).Error("Remote start transaction request failed")
			return
		}

		logrus.WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"connectorID":   connectorID,
			"idTag":         idTag,
			"status":        confirmation.Status,
		}).Info("Remote start transaction request processed")
	}

	req := core.NewRemoteStartTransactionRequest(idTag)
	if connectorID > 0 {
		req.ConnectorId = &connectorID
	}

	return s.centralSystem.OcppServer.RemoteStartTransaction(chargePointID, callback, idTag, func(request *core.RemoteStartTransactionRequest) {
		request.ConnectorId = &connectorID
	})
}

// RemoteStopTransaction sends a remote stop transaction request
func (s *CPMS) RemoteStopTransaction(ctx context.Context, chargePointID string, transactionID int) error {
	callback := func(confirmation *core.RemoteStopTransactionConfirmation, err error) {
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"chargePointID": chargePointID,
				"transactionID": transactionID,
			}).Error("Remote stop transaction request failed")
			return
		}

		logrus.WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"transactionID": transactionID,
			"status":        confirmation.Status,
		}).Info("Remote stop transaction request processed")
	}

	return s.centralSystem.OcppServer.RemoteStopTransaction(chargePointID, callback, transactionID)
}

// TriggerHeartbeat sends a trigger message to request a heartbeat
func (s *CPMS) TriggerHeartbeat(ctx context.Context, chargePointID string) error {
	callback := func(confirmation *remotetrigger.TriggerMessageConfirmation, err error) {
		if err != nil {
			logrus.WithError(err).WithField("chargePointID", chargePointID).Error("Trigger heartbeat request failed")
			return
		}

		logrus.WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"status":        confirmation.Status,
		}).Info("Trigger heartbeat request processed")
	}

	return s.centralSystem.OcppServer.TriggerMessage(chargePointID, callback, core.HeartbeatFeatureName)
}

// TriggerStatusNotification sends a trigger message to request a status notification
func (s *CPMS) TriggerStatusNotification(ctx context.Context, chargePointID string, connectorID int) error {
	callback := func(confirmation *remotetrigger.TriggerMessageConfirmation, err error) {
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"chargePointID": chargePointID,
				"connectorID":   connectorID,
			}).Error("Trigger status notification request failed")
			return
		}

		logrus.WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"connectorID":   connectorID,
			"status":        confirmation.Status,
		}).Info("Trigger status notification request processed")
	}

	return s.centralSystem.OcppServer.TriggerMessage(chargePointID, callback, core.StatusNotificationFeatureName, func(request *remotetrigger.TriggerMessageRequest) {
		if connectorID > 0 {
			request.ConnectorId = &connectorID
		}
	})
}

// GetDiagnostics requests the charge point to upload diagnostics to a remote location
func (s *CPMS) GetDiagnostics(ctx context.Context, chargePointID string, location string, startTime, stopTime time.Time) error {
	callback := func(confirmation *firmware.GetDiagnosticsConfirmation, err error) {
		if err != nil {
			logrus.WithError(err).WithField("chargePointID", chargePointID).Error("Get diagnostics request failed")
			return
		}

		logrus.WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"fileName":      confirmation.FileName,
		}).Info("Get diagnostics request processed")
	}

	return s.centralSystem.OcppServer.GetDiagnostics(chargePointID, callback, location, func(request *firmware.GetDiagnosticsRequest) {
		if !startTime.IsZero() {
			request.StartTime = types.NewDateTime(startTime)
		}
		if !stopTime.IsZero() {
			request.EndTime = types.NewDateTime(stopTime)
		}
	})
}

// UpdateFirmware requests the charge point to download and install new firmware
func (s *CPMS) UpdateFirmware(ctx context.Context, chargePointID string, location string, retrieveDate time.Time) error {
	callback := func(confirmation *firmware.UpdateFirmwareConfirmation, err error) {
		if err != nil {
			logrus.WithError(err).WithField("chargePointID", chargePointID).Error("Update firmware request failed")
			return
		}

		logrus.WithField("chargePointID", chargePointID).Info("Update firmware request processed")
	}

	dt := types.NewDateTime(retrieveDate)
	return s.centralSystem.OcppServer.UpdateFirmware(chargePointID, callback, location, dt)
}

// ClearCache requests the charge point to clear its authorization cache
func (s *CPMS) ClearCache(ctx context.Context, chargePointID string) error {
	callback := func(confirmation *core.ClearCacheConfirmation, err error) {
		if err != nil {
			logrus.WithError(err).WithField("chargePointID", chargePointID).Error("Clear cache request failed")
			return
		}

		logrus.WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"status":        confirmation.Status,
		}).Info("Clear cache request processed")
	}

	return s.centralSystem.OcppServer.ClearCache(chargePointID, callback)
}

// GetConfiguration retrieves the charge point's configuration
func (s *CPMS) GetConfiguration(ctx context.Context, chargePointID string, keys []string) error {
	callback := func(confirmation *core.GetConfigurationConfirmation, err error) {
		if err != nil {
			logrus.WithError(err).WithField("chargePointID", chargePointID).Error("Get configuration request failed")
			return
		}

		logrus.WithFields(logrus.Fields{
			"chargePointID":     chargePointID,
			"configurationKeys": len(confirmation.ConfigurationKey),
			"unknownKeys":       len(confirmation.UnknownKey),
		}).Info("Get configuration request processed")
	}

	return s.centralSystem.OcppServer.GetConfiguration(chargePointID, callback, keys)
}

// ChangeConfiguration changes a configuration key on the charge point
func (s *CPMS) ChangeConfiguration(ctx context.Context, chargePointID string, key string, value string) error {
	callback := func(confirmation *core.ChangeConfigurationConfirmation, err error) {
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"chargePointID": chargePointID,
				"key":           key,
			}).Error("Change configuration request failed")
			return
		}

		logrus.WithFields(logrus.Fields{
			"chargePointID": chargePointID,
			"key":           key,
			"status":        confirmation.Status,
		}).Info("Change configuration request processed")
	}

	return s.centralSystem.OcppServer.ChangeConfiguration(chargePointID, callback, key, value)
}
