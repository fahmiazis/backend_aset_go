-- +goose Up

-- +goose StatementBegin
ALTER TABLE asset_acquisitions
    ADD COLUMN transaction_procurement_id BIGINT UNSIGNED NULL
        COMMENT 'Link ke transaction_procurements untuk tracking per item'
        AFTER transaction_number,
    ADD INDEX idx_asset_acq_procurement_id (transaction_procurement_id),
    ADD CONSTRAINT fk_asset_acq_procurement
        FOREIGN KEY (transaction_procurement_id) REFERENCES transaction_procurements(id)
        ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
ALTER TABLE asset_acquisitions
    DROP FOREIGN KEY fk_asset_acq_procurement,
    DROP INDEX idx_asset_acq_procurement_id,
    DROP COLUMN transaction_procurement_id;
-- +goose StatementEnd