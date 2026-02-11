-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS reservoirs (
    id CHAR(36) PRIMARY KEY,
    no_transaksi VARCHAR(100) NOT NULL UNIQUE,
    kode_plant VARCHAR(50),
    transaksi VARCHAR(50) NOT NULL,
    tipe VARCHAR(50),
    status ENUM('delayed','used','expired') DEFAULT 'delayed',
    branch_id CHAR(36) NULL,
    user_id CHAR(36) NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    
    INDEX idx_no_transaksi (no_transaksi),
    INDEX idx_transaksi (transaksi),
    INDEX idx_status (status),
    INDEX idx_branch_id (branch_id),
    INDEX idx_user_id (user_id),
    INDEX idx_deleted_at (deleted_at),
    
    CONSTRAINT fk_reservoir_branch
        FOREIGN KEY (branch_id)
        REFERENCES branchs(id)
        ON DELETE SET NULL,
    
    CONSTRAINT fk_reservoir_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS reservoirs;
-- +goose StatementEnd