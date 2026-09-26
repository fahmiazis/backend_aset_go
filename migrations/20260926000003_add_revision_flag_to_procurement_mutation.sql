-- +goose Up
-- +goose StatementBegin
-- Penanda revisi per baris, sama dengan yang sudah ada di
-- transaction_disposal_assets (migration 20260920000001).
--
-- Approver menandai item/aset mana yang perlu diperbaiki, lalu transaksinya
-- dikembalikan ke DRAFT. Tanpa penanda per baris, pengaju hanya tahu "ada yang
-- salah" tanpa tahu di bagian mana.
ALTER TABLE transaction_procurements
    ADD COLUMN needs_revision TINYINT(1) NOT NULL DEFAULT 0,
    ADD COLUMN revision_notes TEXT NULL,
    ADD INDEX idx_transaction_procurements_needs_revision (needs_revision);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transaction_mutation_assets
    ADD COLUMN needs_revision TINYINT(1) NOT NULL DEFAULT 0,
    ADD COLUMN revision_notes TEXT NULL,
    ADD INDEX idx_transaction_mutation_assets_needs_revision (needs_revision);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transaction_procurements
    DROP INDEX idx_transaction_procurements_needs_revision,
    DROP COLUMN revision_notes,
    DROP COLUMN needs_revision;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transaction_mutation_assets
    DROP INDEX idx_transaction_mutation_assets_needs_revision,
    DROP COLUMN revision_notes,
    DROP COLUMN needs_revision;
-- +goose StatementEnd
