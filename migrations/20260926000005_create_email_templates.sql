-- +goose Up

-- +goose StatementBegin
-- Template email notifikasi per (jenis transaksi, stage, aksi).
--
-- Dipakai dialog email yang muncul sebelum setiap aksi stage dijalankan.
-- Penerima "To" TIDAK disimpan di sini — dihitung saat itu juga dari user yang
-- punya hak akses di stage berikutnya dan satu cabang dengan pengajuan. Yang
-- disetel hanya subject (dikunci di dialog), body, dan role penerima CC.
--
-- stage = stage transaksi SAAT aksi dilakukan (stage asal), bukan tujuan.
CREATE TABLE email_templates (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_type VARCHAR(50)  NOT NULL COMMENT 'procurement, mutation, disposal',
    stage            VARCHAR(50)  NOT NULL COMMENT 'stage asal saat aksi dilakukan',
    action           ENUM('proceed', 'reject', 'revise', 'cancel') NOT NULL DEFAULT 'proceed',
    subject          VARCHAR(255) NOT NULL,
    body             TEXT         NOT NULL,
    is_active        TINYINT(1)   NOT NULL DEFAULT 1,
    created_by       VARCHAR(100) NULL,
    created_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uq_email_template (transaction_type, stage, action)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose StatementBegin
-- Role yang otomatis masuk CC. Sengaja tanpa FK ke roles: roles.id memakai
-- utf8mb4_general_ci sementara tabel baru unicode_ci, dan pencocokannya memang
-- dilakukan di Go, bukan lewat JOIN.
CREATE TABLE email_template_cc_roles (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    email_template_id BIGINT UNSIGNED NOT NULL,
    role_id           CHAR(36)        NOT NULL,
    created_at        DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uq_email_template_cc_role (email_template_id, role_id),
    CONSTRAINT fk_email_template_cc_template
        FOREIGN KEY (email_template_id) REFERENCES email_templates(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose StatementBegin
-- Jejak setiap email yang dikirim dari dialog. Email yang gagal tetap
-- tersimpan lengkap (penerima, subject, body) supaya bisa dikirim ulang tanpa
-- menyusun ulang isinya.
CREATE TABLE transaction_email_logs (
    id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    email_template_id  BIGINT UNSIGNED NULL,
    transaction_number VARCHAR(100) NOT NULL,
    transaction_type   VARCHAR(50)  NOT NULL,
    stage              VARCHAR(50)  NOT NULL COMMENT 'stage asal saat aksi dilakukan',
    action             VARCHAR(20)  NOT NULL,
    subject            VARCHAR(255) NOT NULL,
    body               MEDIUMTEXT   NOT NULL COMMENT 'HTML final yang dikirim',
    to_emails          TEXT         NOT NULL COMMENT 'JSON array',
    cc_emails          TEXT         NOT NULL COMMENT 'JSON array',
    status             ENUM('SENT', 'FAILED') NOT NULL,
    error_message      TEXT         NULL,
    attempts           INT          NOT NULL DEFAULT 1,
    sent_by            VARCHAR(100) NOT NULL COMMENT 'UUID user yang menjalankan aksi',
    sent_at            DATETIME(3)  NULL,
    created_at         DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at         DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    INDEX idx_email_log_transaction (transaction_number),
    INDEX idx_email_log_status (status),
    INDEX idx_email_log_sent_by (sent_by),
    CONSTRAINT fk_email_log_template
        FOREIGN KEY (email_template_id) REFERENCES email_templates(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose StatementBegin
-- Menu Email Setting di bawah grup Settings, sejajar Attachment Setting.
-- route_path = hasil normalisasi /api/v1/email-templates. CRUD-nya dijaga
-- RequireRole("admin"), jadi permission di bawah hanya menentukan tampil
-- tidaknya menu di sidebar.
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, icon_name, order_index, status, created_at, updated_at)
SELECT UUID(),
       (SELECT id FROM (SELECT id FROM menus WHERE name = 'Settings' AND menu_type = 'group' AND deleted_at IS NULL LIMIT 1) g),
       'Email Setting', 'page',
       '/dashboard/setting-email', '/email-templates',
       'lucide:mail', 1, 'active', NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT * FROM menus) m
    WHERE m.route_path = '/email-templates' AND m.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO menu_permissions (id, menu_id, permission_id, created_at, updated_at)
SELECT UUID(), m.id, p.id, NOW(), NOW()
FROM menus m
JOIN permissions p ON p.value IN ('read', 'write', 'delete')
WHERE m.route_path = '/email-templates'
  AND m.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM (SELECT * FROM menu_permissions) mp
      WHERE mp.menu_id = m.id AND mp.permission_id = p.id
  );
-- +goose StatementEnd

-- +goose StatementBegin
-- Langsung tampil di sidebar admin, supaya tidak perlu assign manual dulu.
INSERT INTO role_menus (id, role_id, menu_id, permissions, created_at, updated_at)
SELECT UUID(), r.id, m.id, '["read","write","delete"]', NOW(), NOW()
FROM roles r
JOIN menus m ON m.route_path = '/email-templates' AND m.deleted_at IS NULL
WHERE r.name = 'admin'
  AND NOT EXISTS (
      SELECT 1 FROM (SELECT * FROM role_menus) rm
      WHERE rm.role_id = r.id AND rm.menu_id = m.id
  );
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DELETE rm FROM role_menus rm
JOIN menus m ON m.id = rm.menu_id
WHERE m.route_path = '/email-templates';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE mp FROM menu_permissions mp
JOIN menus m ON m.id = mp.menu_id
WHERE m.route_path = '/email-templates';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM menus WHERE route_path = '/email-templates';
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_email_logs;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS email_template_cc_roles;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS email_templates;
-- +goose StatementEnd
