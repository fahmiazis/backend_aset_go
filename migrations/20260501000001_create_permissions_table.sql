-- +goose Up
-- +goose StatementBegin
-- Master daftar hak akses.
-- `value` adalah string yang disimpan di role_menus.permissions dan
-- dicocokkan oleh middleware.RequirePermission.
CREATE TABLE IF NOT EXISTS permissions (
    id CHAR(36) PRIMARY KEY,
    value VARCHAR(100) NOT NULL UNIQUE,                    -- mis. "execute_disposal"
    label VARCHAR(100) NOT NULL,                           -- label untuk UI
    description TEXT NULL,
    module VARCHAR(50) NULL,                               -- pengelompokan di UI: basic/procurement/mutation/disposal/...
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_permissions_module (module)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose StatementBegin
-- Seed: seluruh permission yang saat ini dicek RequirePermission(...) di folder routes/,
-- ditambah akses dasar read/write/delete.
INSERT INTO permissions (id, value, label, description, module) VALUES
    (UUID(), 'read',   'Read',   'Melihat data pada menu ini', 'basic'),
    (UUID(), 'write',  'Write',  'Membuat & mengubah data',    'basic'),
    (UUID(), 'delete', 'Delete', 'Menghapus data',             'basic'),

    (UUID(), 'create_transaction', 'Buat Transaksi',   'Membuat & mengubah draft transaksi',        'transaction'),
    (UUID(), 'update_transaction', 'Revisi Transaksi', 'Merevisi transaksi yang sudah berjalan',    'transaction'),
    (UUID(), 'reject_transaction', 'Tolak Transaksi',  'Menolak transaksi berjalan',                'transaction'),

    (UUID(), 'verify_asset',   'Verifikasi Aset', 'Verifikasi item pengadaan',   'procurement'),
    (UUID(), 'process_budget', 'Proses Budget',   'Memproses anggaran pengadaan', 'procurement'),
    (UUID(), 'execute_asset',  'Eksekusi Aset',   'Eksekusi pembuatan aset',      'procurement'),
    (UUID(), 'gr',             'Goods Receipt',   'Input penerimaan barang',      'procurement'),

    (UUID(), 'execute_mutation',  'Eksekusi Mutasi',       'Menjalankan mutasi aset',                 'mutation'),
    (UUID(), 'confirm_receiving', 'Konfirmasi Penerimaan', 'Konfirmasi aset diterima cabang tujuan',  'mutation'),

    (UUID(), 'manage_purchasing',      'Kelola Purchasing', 'Mengisi nilai jual pada stage Purchasing',            'disposal'),
    (UUID(), 'execute_disposal',       'Eksekusi Disposal', 'Menjalankan disposal pada stage Eksekusi',            'disposal'),
    (UUID(), 'manage_finance',         'Kelola Finance',    'Konfirmasi finance pada stage Finance',               'disposal'),
    (UUID(), 'manage_tax',             'Kelola Pajak',      'Konfirmasi pajak pada stage Tax',                     'disposal'),
    (UUID(), 'execute_asset_deletion', 'Penghapusan Aset',  'Konfirmasi penghapusan aset & generate no. dokumen',  'disposal'),

    (UUID(), 'manage_approval',          'Kelola Approval',      'Mengajukan & menyelesaikan flow approval', 'approval'),
    (UUID(), 'upload_attachment',        'Upload Dokumen',       'Mengunggah dokumen transaksi',             'attachment'),
    (UUID(), 'review_attachment',        'Review Dokumen',       'Menyetujui / menolak dokumen',             'attachment'),
    (UUID(), 'manage_attachment_config', 'Konfigurasi Dokumen',  'Mengatur jenis dokumen wajib',             'attachment')
ON DUPLICATE KEY UPDATE
    label       = VALUES(label),
    description = VALUES(description),
    module      = VALUES(module);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS permissions;
-- +goose StatementEnd
