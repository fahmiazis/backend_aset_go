-- +goose Up
-- +goose StatementBegin
-- Nilai pemasukan dan data faktur dipindah dari level transaksi ke level aset.
--
-- Satu pengajuan disposal bisa berisi beberapa aset yang dijual terpisah, jadi
-- uang yang masuk dan fakturnya berbeda per aset. Menyimpannya di level
-- transaksi memaksa satu angka untuk semuanya dan membuat rekonsiliasi per aset
-- tidak mungkin. Polanya disamakan dengan sale_value, yang memang sudah per
-- aset sejak awal.
ALTER TABLE transaction_disposal_assets
    ADD COLUMN income_value DECIMAL(18,2) NULL AFTER sale_value,
    ADD COLUMN invoice_number VARCHAR(100) NULL AFTER income_value,
    ADD COLUMN invoice_date DATE NULL AFTER invoice_number;
-- +goose StatementEnd

-- +goose StatementBegin
-- Pindahkan isi yang terlanjur tersimpan di level transaksi (kalau ada) ke
-- setiap aset aktif milik transaksi tersebut, supaya tidak ada data yang
-- hilang saat kolom lamanya dibuang.
UPDATE transaction_disposal_assets tda
JOIN transactions t ON t.id = tda.transaction_id
SET tda.income_value   = t.income_value,
    tda.invoice_number = t.invoice_number,
    tda.invoice_date   = t.invoice_date
WHERE t.income_value IS NOT NULL
   OR t.invoice_number IS NOT NULL
   OR t.invoice_date IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions
    DROP COLUMN invoice_date,
    DROP COLUMN invoice_number,
    DROP COLUMN income_value;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transactions
    ADD COLUMN income_value DECIMAL(18,2) NULL AFTER sale_value,
    ADD COLUMN invoice_number VARCHAR(100) NULL AFTER income_value,
    ADD COLUMN invoice_date DATE NULL AFTER invoice_number;
-- +goose StatementEnd

-- +goose StatementBegin
-- Balik lagi ke level transaksi: ambil salah satu aset sebagai wakilnya.
UPDATE transactions t
JOIN (
    SELECT transaction_id,
           MAX(income_value)   AS income_value,
           MAX(invoice_number) AS invoice_number,
           MAX(invoice_date)   AS invoice_date
    FROM transaction_disposal_assets
    GROUP BY transaction_id
) a ON a.transaction_id = t.id
SET t.income_value   = a.income_value,
    t.invoice_number = a.invoice_number,
    t.invoice_date   = a.invoice_date;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transaction_disposal_assets
    DROP COLUMN invoice_date,
    DROP COLUMN invoice_number,
    DROP COLUMN income_value;
-- +goose StatementEnd
