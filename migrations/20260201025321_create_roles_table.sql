-- +goose Up
CREATE TABLE roles (
    id CHAR(36) PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    permissions TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Insert default roles
INSERT INTO roles (id, name, description, permissions) VALUES
    (UUID(), 'admin', 'Administrator with full access', '["*"]'),
    (UUID(), 'user', 'Regular user', '["read"]'),
    (UUID(), 'manager', 'Manager with elevated permissions', '["read", "write", "approve"]');

-- +goose Down
DROP TABLE IF EXISTS roles;