-- +goose Up
-- +goose StatementBegin
-- Penanda aset yang diminta direvisi oleh approver.
--
-- Saat approver menekan "Revisi", transaksi memang dikembalikan ke DRAFT
-- seluruhnya, tapi belum tentu semua aset bermasalah. Kolom ini menandai aset
-- mana saja yang boleh disentuh pengaju selama revisi berlangsung; aset yang
-- tidak ditandai dikunci supaya isi pengajuan yang sudah disetujui tidak
-- berubah diam-diam.
--
-- Penanda dibersihkan otomatis saat transaksi disubmit ulang.
ALTER TABLE transaction_disposal_assets
ADD COLUMN needs_revision TINYINT(1) NOT NULL DEFAULT 0 AFTER status,
ADD COLUMN revision_notes TEXT NULL AFTER needs_revision;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_disposal_assets_needs_revision
ON transaction_disposal_assets(transaction_id, needs_revision);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_disposal_assets_needs_revision ON transaction_disposal_assets;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transaction_disposal_assets
DROP COLUMN revision_notes,
DROP COLUMN needs_revision;
-- +goose StatementEnd
