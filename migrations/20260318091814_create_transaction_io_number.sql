-- +goose Up

-- +goose StatementBegin
CREATE TABLE transaction_io_numbers (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_id      BIGINT UNSIGNED NOT NULL,
    transaction_number  VARCHAR(100) NOT NULL,
    branch_code         VARCHAR(50) NOT NULL    COMMENT 'Branch yang punya IO number ini',
    io_number           VARCHAR(50) NOT NULL,
    processed_by        VARCHAR(100) NOT NULL   COMMENT 'UUID PIC Budget',
    processed_at        DATETIME(3) NOT NULL,
    created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uq_transaction_branch_io (transaction_id, branch_code)
        COMMENT 'Satu branch hanya punya 1 IO per transaksi',
    UNIQUE KEY uq_io_number (io_number),
    INDEX idx_tio_transaction_id (transaction_id),
    INDEX idx_tio_transaction_number (transaction_number),
    INDEX idx_tio_branch_code (branch_code),

    CONSTRAINT fk_tio_transaction
        FOREIGN KEY (transaction_id) REFERENCES transactions(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_io_numbers;
-- +goose StatementEnd