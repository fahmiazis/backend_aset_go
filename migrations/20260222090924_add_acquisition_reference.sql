-- +goose Up
-- +goose StatementBegin
ALTER TABLE asset_acquisitions
    ADD COLUMN transaction_id BIGINT UNSIGNED NULL DEFAULT NULL COMMENT 'Reference to transaction header',
    ADD COLUMN transaction_number VARCHAR(100) NULL DEFAULT NULL COMMENT 'Transaction number from main transaction',
    ADD KEY idx_asset_acq_transaction_id (transaction_id),
    ADD KEY idx_asset_acq_transaction_number (transaction_number),
    ADD CONSTRAINT fk_asset_acq_transaction FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE asset_acquisitions
    DROP FOREIGN KEY fk_asset_acq_transaction,
    DROP KEY idx_asset_acq_transaction_id,
    DROP KEY idx_asset_acq_transaction_number,
    DROP COLUMN transaction_id,
    DROP COLUMN transaction_number;
-- +goose StatementEnd
