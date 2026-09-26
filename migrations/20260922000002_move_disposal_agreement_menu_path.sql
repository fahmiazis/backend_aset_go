-- +goose Up
-- +goose StatementBegin
-- Pindahkan path menu kesepakatan disposal dari /dashboard/disposal/agreement
-- ke /dashboard/disposal-agreement.
--
-- Sidebar menandai menu aktif dengan pencocokan awalan
-- (`pathname === path || pathname.startsWith(path + "/")`, lihat
-- organisms/layout/sidebar/content.tsx). Karena path lamanya bersarang di
-- bawah /dashboard/disposal, membuka halaman agreement ikut menyorot menu
-- Disposal — dua menu tersorot sekaligus. Path yang sejajar menghilangkan
-- hubungan awalan itu.
--
-- route_path tidak ikut berubah: itu dipakai middleware.RequirePermission dan
-- tetap /transactions/disposal-agreements.
UPDATE menus
SET path = '/dashboard/disposal-agreement',
    updated_at = NOW()
WHERE path = '/dashboard/disposal/agreement';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE menus
SET path = '/dashboard/disposal/agreement',
    updated_at = NOW()
WHERE path = '/dashboard/disposal-agreement';
-- +goose StatementEnd
