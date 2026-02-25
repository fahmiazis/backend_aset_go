-- +goose Up
-- +goose StatementBegin
CREATE TABLE transaction_procurement_details (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    transaction_procurement_id BIGINT UNSIGNED NOT NULL,
    branch_code VARCHAR(50) NOT NULL,
    quantity INT NOT NULL COMMENT 'Quantity for this specific branch/requester',
    requester_name VARCHAR(255),
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_trans_proc_detail_procurement_id (transaction_procurement_id),
    KEY idx_trans_proc_detail_branch_code (branch_code),
    CONSTRAINT fk_trans_proc_detail_procurement FOREIGN KEY (transaction_procurement_id) REFERENCES transaction_procurements(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Procurement detail breakdown per branch/requester';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_procurement_details;
-- +goose StatementEnd
