package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/balu-dk/go-cpms/config"
	"github.com/balu-dk/go-cpms/internal/db/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// PostgresStore handles database operations
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore initializes a new PostgreSQL connection pool
func NewPostgresStore(cfg *config.Config) (*PostgresStore, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return &PostgresStore{pool: pool}, nil
}

// Close closes the database connection pool
func (s *PostgresStore) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// SaveChargePoint creates or updates a charge point in the database
func (s *PostgresStore) SaveChargePoint(ctx context.Context, cp *models.ChargePoint) error {
	query := `
		INSERT INTO charge_points (
			id, vendor, model, serial_number, firmware_version, 
			last_heartbeat, registration_status, connected_since, is_connected, 
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			vendor = $2,
			model = $3,
			serial_number = $4,
			firmware_version = $5,
			last_heartbeat = $6,
			registration_status = $7,
			connected_since = CASE WHEN charge_points.is_connected = FALSE AND $9 = TRUE THEN $8 ELSE charge_points.connected_since END,
			is_connected = $9,
			updated_at = $11
	`

	now := time.Now()
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = now
	}
	cp.UpdatedAt = now

	_, err := s.pool.Exec(ctx, query,
		cp.ID, cp.Vendor, cp.Model, cp.SerialNumber, cp.FirmwareVersion,
		cp.LastHeartbeat, cp.RegistrationStatus, cp.ConnectedSince, cp.IsConnected,
		cp.CreatedAt, cp.UpdatedAt,
	)
	return err
}

// GetChargePoint retrieves a charge point by its ID
func (s *PostgresStore) GetChargePoint(ctx context.Context, id string) (*models.ChargePoint, error) {
	query := `
		SELECT 
			id, vendor, model, serial_number, firmware_version,
			last_heartbeat, registration_status, connected_since, is_connected,
			created_at, updated_at
		FROM charge_points
		WHERE id = $1
	`

	cp := &models.ChargePoint{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&cp.ID, &cp.Vendor, &cp.Model, &cp.SerialNumber, &cp.FirmwareVersion,
		&cp.LastHeartbeat, &cp.RegistrationStatus, &cp.ConnectedSince, &cp.IsConnected,
		&cp.CreatedAt, &cp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

// GetAllChargePoints retrieves all charge points
func (s *PostgresStore) GetAllChargePoints(ctx context.Context) ([]*models.ChargePoint, error) {
	query := `
		SELECT 
			id, vendor, model, serial_number, firmware_version,
			last_heartbeat, registration_status, connected_since, is_connected,
			created_at, updated_at
		FROM charge_points
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chargePoints []*models.ChargePoint
	for rows.Next() {
		cp := &models.ChargePoint{}
		if err := rows.Scan(
			&cp.ID, &cp.Vendor, &cp.Model, &cp.SerialNumber, &cp.FirmwareVersion,
			&cp.LastHeartbeat, &cp.RegistrationStatus, &cp.ConnectedSince, &cp.IsConnected,
			&cp.CreatedAt, &cp.UpdatedAt,
		); err != nil {
			return nil, err
		}
		chargePoints = append(chargePoints, cp)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return chargePoints, nil
}

// SaveConnector creates or updates a connector
func (s *PostgresStore) SaveConnector(ctx context.Context, connector *models.Connector) error {
	query := `
		INSERT INTO connectors (
			id, charge_point_id, status, error_code, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (charge_point_id, id) DO UPDATE SET
			status = $3,
			error_code = $4,
			updated_at = $6
	`

	now := time.Now()
	if connector.CreatedAt.IsZero() {
		connector.CreatedAt = now
	}
	connector.UpdatedAt = now

	_, err := s.pool.Exec(ctx, query,
		connector.ID, connector.ChargePointID, connector.Status, connector.ErrorCode,
		connector.CreatedAt, connector.UpdatedAt,
	)
	return err
}

// GetConnectors retrieves all connectors for a charge point
func (s *PostgresStore) GetConnectors(ctx context.Context, chargePointID string) ([]*models.Connector, error) {
	query := `
		SELECT 
			id, charge_point_id, status, error_code, created_at, updated_at
		FROM connectors
		WHERE charge_point_id = $1
		ORDER BY id
	`

	rows, err := s.pool.Query(ctx, query, chargePointID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connectors []*models.Connector
	for rows.Next() {
		c := &models.Connector{}
		if err := rows.Scan(
			&c.ID, &c.ChargePointID, &c.Status, &c.ErrorCode,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		connectors = append(connectors, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return connectors, nil
}

// StartTransaction starts a new charging transaction
func (s *PostgresStore) StartTransaction(ctx context.Context, tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (
			id, charge_point_id, connector_id, id_tag, 
			start_time, meter_start, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	now := time.Now()
	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = now
	}
	tx.UpdatedAt = now

	_, err := s.pool.Exec(ctx, query,
		tx.ID, tx.ChargePointID, tx.ConnectorID, tx.IdTag,
		tx.StartTime, tx.MeterStart, tx.Status, tx.CreatedAt, tx.UpdatedAt,
	)
	return err
}

// StopTransaction updates a transaction when it's stopped
func (s *PostgresStore) StopTransaction(ctx context.Context, id int, endTime time.Time, meterStop int) error {
	query := `
		UPDATE transactions
		SET end_time = $1, meter_stop = $2, status = 'Completed', updated_at = $3
		WHERE id = $4
	`

	_, err := s.pool.Exec(ctx, query, endTime, meterStop, time.Now(), id)
	return err
}

// GetTransaction retrieves a transaction by ID
func (s *PostgresStore) GetTransaction(ctx context.Context, id int) (*models.Transaction, error) {
	query := `
		SELECT 
			id, charge_point_id, connector_id, id_tag, 
			start_time, end_time, meter_start, meter_stop, status, 
			created_at, updated_at
		FROM transactions
		WHERE id = $1
	`

	tx := &models.Transaction{}
	var endTime sql.NullTime
	var meterStop sql.NullInt32
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&tx.ID, &tx.ChargePointID, &tx.ConnectorID, &tx.IdTag,
		&tx.StartTime, &endTime, &tx.MeterStart, &meterStop, &tx.Status,
		&tx.CreatedAt, &tx.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if endTime.Valid {
		tx.EndTime = endTime.Time
	}
	if meterStop.Valid {
		tx.MeterStop = int(meterStop.Int32)
	}

	return tx, nil
}

// LogOCPPMessage logs an OCPP message to the database
func (s *PostgresStore) LogOCPPMessage(ctx context.Context, msg *models.OCPPMessage) error {
	query := `
		INSERT INTO ocpp_messages (
			charge_point_id, message_type, action, request_id, payload, direction, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	payload, err := json.Marshal(msg.Payload)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal OCPP message payload")
		payload = []byte("{}")
	}

	_, err = s.pool.Exec(ctx, query,
		msg.ChargePointID, msg.MessageType, msg.Action, msg.RequestID, payload, msg.Direction, msg.Timestamp,
	)
	return err
}

// SaveMeterValue saves a meter reading
func (s *PostgresStore) SaveMeterValue(ctx context.Context, mv *models.MeterValue) error {
	query := `
		INSERT INTO meter_values (
			transaction_id, charge_point_id, connector_id, timestamp, value, unit, measurand, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := s.pool.Exec(ctx, query,
		mv.TransactionID, mv.ChargePointID, mv.ConnectorID, mv.Timestamp,
		mv.Value, mv.Unit, mv.Measurand, time.Now(),
	)
	return err
}

// UpdateChargePointConnection updates the connection status of a charge point
func (s *PostgresStore) UpdateChargePointConnection(ctx context.Context, id string, connected bool) error {
	var query string
	var args []interface{}
	now := time.Now()

	if connected {
		query = `
			UPDATE charge_points
			SET is_connected = true, connected_since = $1, updated_at = $2
			WHERE id = $3
		`
		args = []interface{}{now, now, id}
	} else {
		query = `
			UPDATE charge_points
			SET is_connected = false, updated_at = $1
			WHERE id = $2
		`
		args = []interface{}{now, id}
	}

	_, err := s.pool.Exec(ctx, query, args...)
	return err
}

// UpdateHeartbeat updates the last heartbeat time of a charge point
func (s *PostgresStore) UpdateHeartbeat(ctx context.Context, id string) error {
	query := `
		UPDATE charge_points
		SET last_heartbeat = $1, updated_at = $1
		WHERE id = $2
	`

	now := time.Now()
	_, err := s.pool.Exec(ctx, query, now, id)
	return err
}
