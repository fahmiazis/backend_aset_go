-- +goose Up
-- +goose StatementBegin
INSERT INTO stock_opname_draft_items
    (id, transaction_id, transaction_number, asset_id, asset_number,
     physical_status, `condition`, asset_status, notes, created_at, updated_at)
SELECT tso.id, tso.transaction_id, tso.transaction_number, tso.asset_id, tso.asset_number,
       tso.physical_status, tso.`condition`, tso.asset_status, tso.notes, tso.created_at, tso.updated_at
FROM transaction_stock_opnames tso
JOIN transactions t ON t.id = tso.transaction_id
WHERE t.current_stage = 'DRAFT';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE tso FROM transaction_stock_opnames tso
JOIN transactions t ON t.id = tso.transaction_id
WHERE t.current_stage = 'DRAFT';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
INSERT INTO transaction_stock_opnames
    (id, transaction_id, transaction_number, asset_id, asset_number,
     physical_status, `condition`, asset_status, notes, created_at, updated_at)
SELECT sdi.id, sdi.transaction_id, sdi.transaction_number, sdi.asset_id, sdi.asset_number,
       sdi.physical_status, sdi.`condition`, sdi.asset_status, sdi.notes, sdi.created_at, sdi.updated_at
FROM stock_opname_draft_items sdi
JOIN transactions t ON t.id = sdi.transaction_id
WHERE t.current_stage = 'DRAFT';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE sdi FROM stock_opname_draft_items sdi
JOIN transactions t ON t.id = sdi.transaction_id
WHERE t.current_stage = 'DRAFT';
-- +goose StatementEnd
