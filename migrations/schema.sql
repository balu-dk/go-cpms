-- Schema for CPMS PostgreSQL database

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Charge Points table
CREATE TABLE IF NOT EXISTS charge_points (
    id VARCHAR(100) PRIMARY KEY,
    vendor VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    serial_number VARCHAR(100),
    firmware_version VARCHAR(100),
    last_heartbeat TIMESTAMP WITH TIME ZONE,
    registration_status VARCHAR(20) NOT NULL,
    connected_since TIMESTAMP WITH TIME ZONE,
    is_connected BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Connectors table
CREATE TABLE IF NOT EXISTS connectors (
    id INTEGER NOT NULL,
    charge_point_id VARCHAR(100) NOT NULL REFERENCES charge_points(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL,
    error_code VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (charge_point_id, id)
);

-- Transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id INTEGER PRIMARY KEY,
    charge_point_id VARCHAR(100) NOT NULL REFERENCES charge_points(id),
    connector_id INTEGER NOT NULL,
    id_tag VARCHAR(100) NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    meter_start INTEGER NOT NULL,
    meter_stop INTEGER,
    status VARCHAR(20) NOT NULL, -- InProgress, Completed, Stopped
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT transaction_connector_fk FOREIGN KEY (charge_point_id, connector_id) REFERENCES connectors(charge_point_id, id)
);

-- OCPP Messages table for logging
CREATE TABLE IF NOT EXISTS ocpp_messages (
    id SERIAL PRIMARY KEY,
    charge_point_id VARCHAR(100) NOT NULL REFERENCES charge_points(id),
    message_type VARCHAR(20) NOT NULL, -- Request or Response
    action VARCHAR(100) NOT NULL,
    request_id VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    direction VARCHAR(20) NOT NULL, -- Inbound or Outbound
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX IF NOT EXISTS ocpp_messages_cp_id_idx ON ocpp_messages(charge_point_id);
CREATE INDEX IF NOT EXISTS ocpp_messages_timestamp_idx ON ocpp_messages(timestamp);

-- Meter Values table
CREATE TABLE IF NOT EXISTS meter_values (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER REFERENCES transactions(id),
    charge_point_id VARCHAR(100) NOT NULL REFERENCES charge_points(id),
    connector_id INTEGER NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    unit VARCHAR(10) NOT NULL,
    measurand VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT meter_values_connector_fk FOREIGN KEY (charge_point_id, connector_id) REFERENCES connectors(charge_point_id, id)
);
CREATE INDEX IF NOT EXISTS meter_values_transaction_idx ON meter_values(transaction_id);
CREATE INDEX IF NOT EXISTS meter_values_cp_connector_idx ON meter_values(charge_point_id, connector_id);

-- Create indexes
CREATE INDEX IF NOT EXISTS charge_points_connected_idx ON charge_points(is_connected);
CREATE INDEX IF NOT EXISTS transactions_status_idx ON transactions(status);