-- +goose Up
-- +goose StatementBegin
-- Data yang wajib diisi di stage akhir disposal SELL.
--
-- income_value diisi finance: uang yang benar-benar masuk, bisa berbeda dengan
-- sale_value yang disepakati purchasing (potongan, pembatalan sebagian, dsb).
-- Karena itu disimpan terpisah, bukan menimpa sale_value.
--
-- invoice_number / invoice_date diisi tim pajak sebagai rujukan faktur.
-- Semuanya nullable karena transaksi DISPOSE tidak melewati stage ini, dan
-- transaksi SELL yang sudah berjalan sebelum migration ini tidak punya isinya.
ALTER TABLE transactions
    ADD COLUMN income_value DECIMAL(18,2) NULL AFTER sale_value,
    ADD COLUMN invoice_number VARCHAR(100) NULL AFTER income_value,
    ADD COLUMN invoice_date DATE NULL AFTER invoice_number;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transactions
    DROP COLUMN invoice_date,
    DROP COLUMN invoice_number,
    DROP COLUMN income_value;
-- +goose StatementEnd
