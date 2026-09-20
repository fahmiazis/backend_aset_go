-- +goose Up
-- +goose StatementBegin
-- Perbaikan: menu yang dibuat SETELAH migrasi 20260501000002 tidak pernah
-- mendapat baris di menu_permissions, karena seed di migrasi itu hanya
-- menjangkau menu yang sudah ada saat itu dan CreateMenu belum mengisinya.
--
-- Akibatnya menu tersebut tidak punya satu pun pilihan permission di matriks
-- hak akses role, sehingga TIDAK BISA di-assign sama sekali. Paling terasa
-- pada menu grup yang baru dibuat: grupnya tidak pernah muncul di sidebar
-- dan percobaan meng-assign-nya selalu gagal tanpa pesan error.
--
-- Migrasi ini melengkapi hak akses dasar (read/write/delete) untuk SEMUA menu
-- yang belum punya. Aman dijalankan berulang karena memakai INSERT IGNORE
-- dan UNIQUE (menu_id, permission_id).
INSERT IGNORE INTO menu_permissions (id, menu_id, permission_id)
SELECT UUID(), m.id, p.id
FROM menus m
CROSS JOIN permissions p
WHERE p.module = 'basic'
  AND m.deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Tidak ada rollback yang aman: tidak mungkin membedakan baris hasil backfill
-- ini dari pemetaan yang memang sengaja dibuat pengguna.
SELECT 1;
-- +goose StatementEnd
