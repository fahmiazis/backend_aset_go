-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS approval_flows (
    id CHAR(36) PRIMARY KEY,
    flow_code VARCHAR(50) NOT NULL UNIQUE,
    flow_name VARCHAR(100) NOT NULL,
    type ENUM('it','non-it','all') DEFAULT 'all',
    category ENUM('budget','non-budget','return','all') DEFAULT 'all',
    approval_way ENUM('sequential','parallel','conditional') DEFAULT 'sequential',
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    
    INDEX idx_flow_code (flow_code),
    INDEX idx_is_active (is_active),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS approval_flows;
-- +goose StatementEnd