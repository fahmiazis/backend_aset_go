-- +goose Up
-- +goose StatementBegin
-- Menu bertipe permission untuk endpoint per stage disposal.
--
-- middleware.RequirePermission MENOLAK request kalau route-nya tidak punya
-- baris menu ("Access denied: route not registered in menu"). Menu untuk
-- purchasing, approval, execute, finance, tax, asset-deletion, dan reject
-- belum pernah dibuat — akibatnya seluruh aksi disposal setelah DRAFT selalu
-- 403 untuk SEMUA user, termasuk admin.
--
-- route_path harus sama dengan hasil normalisasi path request: buang /api/v1,
-- lewati segment id, ambil maksimal 3 segment.
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, order_index, status, created_at, updated_at)
SELECT UUID(), NULL, src.name, 'permission', src.path, src.route_path, 0, 'active', NOW(), NOW()
FROM (
              SELECT 'Disposal Purchasing'         AS name, '/dashboard/disposal/purchasing'     AS path, '/transactions/disposal/purchasing'         AS route_path
    UNION ALL SELECT 'Disposal Approval Request',        '/dashboard/disposal/approval-request',       '/transactions/disposal/approval-request'
    UNION ALL SELECT 'Disposal Approval Agreement',      '/dashboard/disposal/approval-agreement',     '/transactions/disposal/approval-agreement'
    UNION ALL SELECT 'Disposal Execute',                 '/dashboard/disposal/execute',                '/transactions/disposal/execute'
    UNION ALL SELECT 'Disposal Finance',                 '/dashboard/disposal/finance',                '/transactions/disposal/finance'
    UNION ALL SELECT 'Disposal Tax',                     '/dashboard/disposal/tax',                    '/transactions/disposal/tax'
    UNION ALL SELECT 'Disposal Asset Deletion',          '/dashboard/disposal/asset-deletion',         '/transactions/disposal/asset-deletion'
    UNION ALL SELECT 'Disposal Reject',                  '/dashboard/disposal/reject',                 '/transactions/disposal/reject'
) src
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT route_path, deleted_at FROM menus) m
    WHERE m.route_path = src.route_path AND m.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Pilihan permission yang muncul di matriks hak akses untuk tiap menu di atas
INSERT INTO menu_permissions (id, menu_id, permission_id, created_at, updated_at)
SELECT UUID(), mn.id, p.id, NOW(), NOW()
FROM (
              SELECT '/transactions/disposal/purchasing'         AS route_path, 'manage_purchasing'      AS perm
    UNION ALL SELECT '/transactions/disposal/approval-request',        'manage_approval'
    UNION ALL SELECT '/transactions/disposal/approval-agreement',      'manage_approval'
    UNION ALL SELECT '/transactions/disposal/execute',                 'execute_disposal'
    UNION ALL SELECT '/transactions/disposal/finance',                 'manage_finance'
    UNION ALL SELECT '/transactions/disposal/tax',                     'manage_tax'
    UNION ALL SELECT '/transactions/disposal/asset-deletion',          'execute_asset_deletion'
    UNION ALL SELECT '/transactions/disposal/reject',                  'reject_transaction'
) map
JOIN menus mn ON mn.route_path = map.route_path AND mn.deleted_at IS NULL
JOIN permissions p ON p.value = map.perm
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT menu_id, permission_id FROM menu_permissions) mp
    WHERE mp.menu_id = mn.id AND mp.permission_id = p.id
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE mp FROM menu_permissions mp
JOIN menus m ON m.id = mp.menu_id
WHERE m.route_path IN (
    '/transactions/disposal/purchasing',
    '/transactions/disposal/approval-request',
    '/transactions/disposal/approval-agreement',
    '/transactions/disposal/execute',
    '/transactions/disposal/finance',
    '/transactions/disposal/tax',
    '/transactions/disposal/asset-deletion',
    '/transactions/disposal/reject'
);
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM menus WHERE route_path IN (
    '/transactions/disposal/purchasing',
    '/transactions/disposal/approval-request',
    '/transactions/disposal/approval-agreement',
    '/transactions/disposal/execute',
    '/transactions/disposal/finance',
    '/transactions/disposal/tax',
    '/transactions/disposal/asset-deletion',
    '/transactions/disposal/reject'
);
-- +goose StatementEnd
