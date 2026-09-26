-- +goose Up
-- +goose StatementBegin
-- Perbaiki status transaksi yang telanjur salah.
--
-- stageToStatus dulu hanya memetakan stage procurement, sehingga seluruh stage
-- disposal dan mutasi di luar DRAFT/APPROVAL/FINISHED jatuh ke fallback dan
-- statusnya tertulis DRAFT walaupun transaksinya sudah berjalan jauh.
--
-- Pemetaannya sama persis dengan stageToStatus setelah diperbaiki. Stage yang
-- tidak dikenal (mis. "SUBMITTED" dari alur lama) sengaja tidak disentuh —
-- tidak ada padanan yang bisa dipastikan benar untuk itu.
UPDATE transactions
SET status = CASE current_stage
    WHEN 'DRAFT'                THEN 'DRAFT'
    WHEN 'ASSET_VERIFICATION'   THEN 'PENDING'
    WHEN 'APPROVAL'             THEN 'PENDING'
    WHEN 'APPROVAL_REQUEST'     THEN 'PENDING'
    WHEN 'APPROVAL_AGREEMENT'   THEN 'PENDING'
    WHEN 'PROCESS_BUDGET'       THEN 'PROCESSING'
    WHEN 'EXECUTE_ASET'         THEN 'PROCESSING'
    WHEN 'GR'                   THEN 'PROCESSING'
    WHEN 'PURCHASING'           THEN 'PROCESSING'
    WHEN 'EXECUTE'              THEN 'PROCESSING'
    WHEN 'FINANCE'              THEN 'PROCESSING'
    WHEN 'TAX'                  THEN 'PROCESSING'
    WHEN 'ASSET_DELETION'       THEN 'PROCESSING'
    WHEN 'MUTATION_RECEIVING'   THEN 'PROCESSING'
    WHEN 'EXECUTE_MUTATION'     THEN 'PROCESSING'
    WHEN 'FINISHED'             THEN 'APPROVED'
    WHEN 'REJECTED'             THEN 'REJECTED'
    WHEN 'CANCELLED'            THEN 'CANCELLED'
    ELSE status
END
WHERE current_stage IN (
    'DRAFT', 'ASSET_VERIFICATION', 'APPROVAL', 'APPROVAL_REQUEST',
    'APPROVAL_AGREEMENT', 'PROCESS_BUDGET', 'EXECUTE_ASET', 'GR',
    'PURCHASING', 'EXECUTE', 'FINANCE', 'TAX', 'ASSET_DELETION',
    'MUTATION_RECEIVING', 'EXECUTE_MUTATION',
    'FINISHED', 'REJECTED', 'CANCELLED'
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Tidak dibalik: nilai status sebelumnya memang salah, dan nilai aslinya per
-- baris tidak tersimpan di mana pun.
SELECT 1;
-- +goose StatementEnd
