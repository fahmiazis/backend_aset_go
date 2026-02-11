-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS approval_signatures (
    id CHAR(36) PRIMARY KEY,
    transaction_number VARCHAR(100) NOT NULL,
    transaction_type VARCHAR(50) NOT NULL,
    user_id CHAR(36) NOT NULL,
    role_id CHAR(36) NULL,
    step_role ENUM('creator','reviewer','approver','receiver') NOT NULL,
    signature_path VARCHAR(255) NULL,
    signed_at TIMESTAMP NOT NULL,
    status ENUM('signed','rejected') DEFAULT 'signed',
    notes TEXT NULL,
    ip_address VARCHAR(45) NULL,
    user_agent VARCHAR(255) NULL,
    structure VARCHAR(100) NULL,
    is_recent BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    
    INDEX idx_transaction (transaction_number, transaction_type),
    INDEX idx_user_id (user_id),
    INDEX idx_signed_at (signed_at),
    INDEX idx_is_recent (is_recent),
    INDEX idx_deleted_at (deleted_at),
    
    CONSTRAINT fk_app_sig_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,
    
    CONSTRAINT fk_app_sig_role
        FOREIGN KEY (role_id)
        REFERENCES roles(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS approval_signatures;
-- +goose StatementEnd