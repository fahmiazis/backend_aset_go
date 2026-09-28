-- +goose Up
-- Menu bertipe permission untuk endpoint stage procurement & mutation.
--
-- Dua masalah yang ditutup:
--
-- 1. Enam menu stage procurement (submit, verify, approval, process-budget,
--    execute, gr) menempel ke grup "Procurement" lama yang sudah di-soft-delete
--    (17 Mei 2026). GetAllMenus hanya menelusuri dari induk yang hidup, jadi
--    keenamnya tidak pernah tampil di /dashboard/menu maupun matriks hak akses
--    role — tidak bisa di-assign. Tipenya juga masih `page` padahal tidak ada
--    halamannya. Dipindah ke bawah "Procurement list" dan dijadikan `permission`.
--
-- 2. Endpoint yang belum punya baris menu sama sekali, sehingga 403 "route not
--    registered" untuk SEMUA user, termasuk admin:
--      POST /transactions/procurement/reject      reject_transaction
--      PUT  /transactions/procurement/revise      update_transaction
--      POST /transactions/mutation/reject         reject_transaction
--      POST /transactions/mutation/attachments/*  upload_attachment | create_transaction | review_attachment
--
-- 3. Menu "Procurement verify" yang bentrok route_path dengan "Procurement
--    transaction" dihapus (detail di bagian bawah).
--
-- route_path = hasil normalisasi RequirePermission: buang /api/v1, lewati
-- segment id, ambil maksimal 3 segment.
--
-- Idempotent: UPDATE berbasis route_path, INSERT dijaga NOT EXISTS. Aman
-- dijalankan ulang, termasuk kalau sebagian menu sudah dibuat manual lewat UI.
--
-- Hak akses role TIDAK diberikan di sini — assign lewat /dashboard/role/:id.

-- +goose StatementBegin
-- Induk menu procurement: "Procurement list" (induk menu permission
-- /transactions/procurement yang masih hidup). Fallback ke menu Procurement
-- level atas kalau tidak ketemu.
SET @procurement_parent := COALESCE(
    (SELECT p.id FROM menus c
     JOIN menus p ON p.id = c.parent_id AND p.deleted_at IS NULL
     WHERE c.route_path = '/transactions/procurement' AND c.deleted_at IS NULL
     ORDER BY p.order_index LIMIT 1),
    (SELECT id FROM menus
     WHERE parent_id IS NULL AND path = '/dashboard/procurement' AND deleted_at IS NULL
     ORDER BY order_index LIMIT 1)
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Induk menu mutation: "Mutation List" (induk menu permission Mutation Draft).
SET @mutation_parent := COALESCE(
    (SELECT p.id FROM menus c
     JOIN menus p ON p.id = c.parent_id AND p.deleted_at IS NULL
     WHERE c.route_path = '/transactions/mutation/draft' AND c.deleted_at IS NULL
     ORDER BY p.order_index LIMIT 1),
    (SELECT id FROM menus
     WHERE parent_id IS NULL AND path = '/dashboard/mutation' AND deleted_at IS NULL
     ORDER BY order_index LIMIT 1)
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Menu stage procurement yang sudah ada: pindah induk, jadikan permission,
-- rapikan nama (dua-duanya dulu "Asset Verification") dan path (execute & submit
-- dulu memakai path milik menu lain). id tetap → role_menus yang ada tidak hilang.
UPDATE menus m
JOIN (
              SELECT '/transactions/procurement/submit'         AS route_path, 'Procurement Submit'         AS name, '/dashboard/procurement/submit'         AS path, 1 AS ord
    UNION ALL SELECT '/transactions/procurement/verify',                'Procurement Verify Asset',          '/dashboard/procurement/verify',                2
    UNION ALL SELECT '/transactions/procurement/approval',              'Procurement Approval',              '/dashboard/procurement/approval',              3
    UNION ALL SELECT '/transactions/procurement/process-budget',        'Procurement Process Budget',        '/dashboard/procurement/process-budget',        4
    UNION ALL SELECT '/transactions/procurement/execute',               'Procurement Execute',               '/dashboard/procurement/execute',               5
    UNION ALL SELECT '/transactions/procurement/gr',                    'Procurement Goods Receipt',         '/dashboard/procurement/gr',                    6
) src ON src.route_path = m.route_path
SET m.parent_id   = COALESCE(@procurement_parent, m.parent_id),
    m.menu_type   = 'permission',
    m.name        = src.name,
    m.path        = src.path,
    m.order_index = src.ord,
    m.updated_at  = NOW()
WHERE m.deleted_at IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- Menu baru untuk endpoint yang belum terdaftar
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, order_index, status, created_at, updated_at)
SELECT UUID(), src.parent_id, src.name, 'permission', src.path, src.route_path, src.ord, 'active', NOW(), NOW()
FROM (
              SELECT @procurement_parent AS parent_id, 'Procurement Reject' AS name, '/dashboard/procurement/reject' AS path, '/transactions/procurement/reject' AS route_path, 7 AS ord
    UNION ALL SELECT @procurement_parent, 'Procurement Revise',      '/dashboard/procurement/revise',      '/transactions/procurement/revise',       8
    UNION ALL SELECT @mutation_parent,    'Mutation Reject',         '/dashboard/mutation/reject',         '/transactions/mutation/reject',          4
    UNION ALL SELECT @mutation_parent,    'Mutation Attachment',     '/dashboard/mutation/attachments',    '/transactions/mutation/attachments',     5
) src
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT route_path, deleted_at FROM menus) m
    WHERE m.route_path = src.route_path AND m.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Pilihan permission di matriks hak akses. Mencakup juga menu lama, supaya
-- checkbox-nya pasti ada walau pemetaan sebelumnya belum lengkap.
INSERT INTO menu_permissions (id, menu_id, permission_id, created_at, updated_at)
SELECT UUID(), mn.id, p.id, NOW(), NOW()
FROM (
              SELECT '/transactions/procurement/submit'          AS route_path, 'create_transaction' AS perm
    UNION ALL SELECT '/transactions/procurement/verify',                'verify_asset'
    UNION ALL SELECT '/transactions/procurement/approval',              'manage_approval'
    UNION ALL SELECT '/transactions/procurement/process-budget',        'process_budget'
    UNION ALL SELECT '/transactions/procurement/execute',               'execute_asset'
    UNION ALL SELECT '/transactions/procurement/gr',                    'create_transaction'
    UNION ALL SELECT '/transactions/procurement/gr',                    'gr'
    UNION ALL SELECT '/transactions/procurement/reject',                'reject_transaction'
    UNION ALL SELECT '/transactions/procurement/revise',                'update_transaction'
    UNION ALL SELECT '/transactions/mutation/draft',                    'create_transaction'
    UNION ALL SELECT '/transactions/mutation/approval',                 'manage_approval'
    UNION ALL SELECT '/transactions/mutation/confirm-receiving',        'confirm_receiving'
    UNION ALL SELECT '/transactions/mutation/confirm-receiving',        'create_transaction'
    UNION ALL SELECT '/transactions/mutation/execute',                  'execute_mutation'
    UNION ALL SELECT '/transactions/mutation/reject',                   'reject_transaction'
    UNION ALL SELECT '/transactions/mutation/attachments',              'upload_attachment'
    UNION ALL SELECT '/transactions/mutation/attachments',              'create_transaction'
    UNION ALL SELECT '/transactions/mutation/attachments',              'review_attachment'
) map
JOIN menus mn ON mn.route_path = map.route_path AND mn.deleted_at IS NULL
JOIN permissions p ON p.value = map.perm
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT menu_id, permission_id FROM menu_permissions) mp
    WHERE mp.menu_id = mn.id AND mp.permission_id = p.id
);
-- +goose StatementEnd

-- ─── Hapus menu "Procurement verify" ─────────────────────────────────────────
-- Menu ini berbagi route_path /transactions/procurement dengan "Procurement
-- transaction". RequirePermission memakai First() tanpa urutan, jadi kalau yang
-- terambil menu ini (isinya cuma verify_asset), POST /transactions/procurement
-- (create, butuh create_transaction) bisa 403. Endpoint verify sendiri
-- ber-route /transactions/procurement/verify, jadi menu ini tidak pernah
-- memberi akses verify apa pun. Assignment verify_asset yang ada (pic asset)
-- dipindah dulu ke menu verify yang benar supaya maksudnya tetap terjaga.

-- +goose StatementBegin
-- Role yang sudah punya baris di menu verify: tambahkan verify_asset
UPDATE role_menus target
JOIN menus tm ON tm.id = target.menu_id AND tm.route_path = '/transactions/procurement/verify' AND tm.deleted_at IS NULL
JOIN role_menus src ON src.role_id = target.role_id
JOIN menus sm ON sm.id = src.menu_id AND sm.route_path = '/transactions/procurement'
             AND sm.name = 'Procurement verify' AND sm.deleted_at IS NULL
SET target.permissions = JSON_ARRAY_APPEND(target.permissions, '$', 'verify_asset'),
    target.updated_at  = NOW()
WHERE JSON_SEARCH(src.permissions, 'one', 'verify_asset') IS NOT NULL
  AND JSON_SEARCH(target.permissions, 'one', 'verify_asset') IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- Role yang belum punya baris di menu verify: buat baru
INSERT INTO role_menus (id, role_id, menu_id, permissions, created_at, updated_at)
SELECT UUID(), src.role_id, tm.id, '["verify_asset"]', NOW(), NOW()
FROM role_menus src
JOIN menus sm ON sm.id = src.menu_id AND sm.route_path = '/transactions/procurement'
             AND sm.name = 'Procurement verify' AND sm.deleted_at IS NULL
JOIN menus tm ON tm.route_path = '/transactions/procurement/verify' AND tm.deleted_at IS NULL
WHERE JSON_SEARCH(src.permissions, 'one', 'verify_asset') IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM (SELECT role_id, menu_id FROM role_menus) x
      WHERE x.role_id = src.role_id AND x.menu_id = tm.id
  );
-- +goose StatementEnd

-- +goose StatementBegin
DELETE rm FROM role_menus rm
JOIN menus m ON m.id = rm.menu_id
WHERE m.route_path = '/transactions/procurement' AND m.name = 'Procurement verify';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE mp FROM menu_permissions mp
JOIN menus m ON m.id = mp.menu_id
WHERE m.route_path = '/transactions/procurement' AND m.name = 'Procurement verify';
-- +goose StatementEnd

-- +goose StatementBegin
-- Soft delete, sama dengan hapus menu lewat UI (GORM DeletedAt)
UPDATE menus
SET deleted_at = NOW(), updated_at = NOW()
WHERE route_path = '/transactions/procurement' AND name = 'Procurement verify'
  AND deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- Hanya menghapus menu yang DIBUAT migrasi ini. Pemindahan induk menu stage
-- procurement tidak dikembalikan: induk lamanya sudah terhapus, mengembalikannya
-- berarti menyembunyikan menu itu lagi dari /dashboard/menu. Menu "Procurement
-- verify" juga tidak dipulihkan: memulihkannya membuka lagi bentrok route_path
-- dengan "Procurement transaction", dan verify_asset milik role sudah
-- berfungsi di menu /transactions/procurement/verify.

-- +goose StatementBegin
DELETE rm FROM role_menus rm
JOIN menus m ON m.id = rm.menu_id
WHERE m.route_path IN (
    '/transactions/procurement/reject',
    '/transactions/procurement/revise',
    '/transactions/mutation/reject',
    '/transactions/mutation/attachments'
);
-- +goose StatementEnd

-- +goose StatementBegin
DELETE mp FROM menu_permissions mp
JOIN menus m ON m.id = mp.menu_id
WHERE m.route_path IN (
    '/transactions/procurement/reject',
    '/transactions/procurement/revise',
    '/transactions/mutation/reject',
    '/transactions/mutation/attachments'
);
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM menus WHERE menu_type = 'permission' AND route_path IN (
    '/transactions/procurement/reject',
    '/transactions/procurement/revise',
    '/transactions/mutation/reject',
    '/transactions/mutation/attachments'
);
-- +goose StatementEnd
