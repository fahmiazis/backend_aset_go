-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS transaction_approvals (
    id CHAR(36) PRIMARY KEY,
    flow_id CHAR(36) NOT NULL,
    flow_step_id CHAR(36) NOT NULL,
    transaction_number VARCHAR(100) NOT NULL,
    transaction_type VARCHAR(50) NOT NULL,
    approver_user_id CHAR(36) NULL,
    approver_role_id CHAR(36) NULL,
    status ENUM('pending','approved','rejected','skipped') DEFAULT 'pending',
    status_view ENUM('visible','hidden') DEFAULT 'visible',
    approved_at TIMESTAMP NULL,
    approved_by CHAR(36) NULL,
    rejected_at TIMESTAMP NULL,
    rejected_by CHAR(36) NULL,
    notes TEXT NULL,
    metadata JSON NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    
    INDEX idx_transaction (transaction_number, transaction_type),
    INDEX idx_flow_id (flow_id),
    INDEX idx_flow_step_id (flow_step_id),
    INDEX idx_approver_user_id (approver_user_id),
    INDEX idx_approver_role_id (approver_role_id),
    INDEX idx_status (status),
    INDEX idx_status_view (status_view),
    INDEX idx_deleted_at (deleted_at),
    
    CONSTRAINT fk_trans_app_flow
        FOREIGN KEY (flow_id)
        REFERENCES approval_flows(id)
        ON DELETE CASCADE,
    
    CONSTRAINT fk_trans_app_flow_step
        FOREIGN KEY (flow_step_id)
        REFERENCES approval_flow_steps(id)
        ON DELETE CASCADE,
    
    CONSTRAINT fk_trans_app_approver_user
        FOREIGN KEY (approver_user_id)
        REFERENCES users(id)
        ON DELETE SET NULL,
    
    CONSTRAINT fk_trans_app_approver_role
        FOREIGN KEY (approver_role_id)
        REFERENCES roles(id)
        ON DELETE SET NULL,
    
    CONSTRAINT fk_trans_app_approved_by
        FOREIGN KEY (approved_by)
        REFERENCES users(id)
        ON DELETE SET NULL,
    
    CONSTRAINT fk_trans_app_rejected_by
        FOREIGN KEY (rejected_by)
        REFERENCES users(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_approvals;
-- +goose StatementEnd