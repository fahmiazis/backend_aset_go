-- +goose Up
-- +goose StatementBegin
-- Jenis menu, memisahkan "yang tampil di sidebar" dari "yang hanya untuk hak akses".
--
--   page       : menu biasa, punya halaman, tampil di sidebar
--   group      : wadah/grup di sidebar, tidak punya halaman
--   permission : TIDAK tampil di sidebar. Hanya wadah pemetaan route_path → permission
--                untuk middleware.RequirePermission. Dipakai untuk endpoint yang tidak
--                punya halaman sendiri, mis. /transactions/mutation/draft
--
-- Sebelumnya satu-satunya cara menyembunyikan menu adalah status='inactive', tapi itu
-- salah kaprah: RequirePermission tidak memfilter status (akses tetap jalan) sementara
-- GetRoleMenus memfilternya (hak aksesnya hilang dari UI dan bisa tercabut tanpa sengaja).
ALTER TABLE menus
ADD COLUMN menu_type ENUM('page','group','permission') NOT NULL DEFAULT 'page' AFTER name;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_menus_menu_type ON menus(menu_type);
-- +goose StatementEnd

-- +goose StatementBegin
-- Backfill: menu tanpa path tidak bisa dibuka sebagai halaman, jadi ia sebuah grup.
UPDATE menus
SET menu_type = 'group'
WHERE (path IS NULL OR path = '')
  AND deleted_at IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- Menu yang selama ini disembunyikan lewat status='inactive' padahal punya route_path
-- kemungkinan besar memang hanya untuk hak akses. Ditandai sekaligus diaktifkan kembali,
-- karena sekarang penyembunyiannya ditangani menu_type, bukan status.
UPDATE menus
SET menu_type = 'permission',
    status    = 'active'
WHERE status = 'inactive'
  AND route_path IS NOT NULL
  AND route_path <> ''
  AND deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_menus_menu_type ON menus;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE menus DROP COLUMN menu_type;
-- +goose StatementEnd
