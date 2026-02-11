-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS approval_flow_steps (
    id CHAR(36) PRIMARY KEY,
    flow_id CHAR(36) NOT NULL,
    step_order INT NOT NULL,
    step_name VARCHAR(100) NOT NULL,
    step_role ENUM('creator','reviewer','approver','receiver') NOT NULL,
    role_id CHAR(36) NULL,
    branch_id CHAR(36) NULL,
    structure VARCHAR(100) NULL,
    is_required BOOLEAN DEFAULT TRUE,
    can_skip BOOLEAN DEFAULT FALSE,
    is_visible BOOLEAN DEFAULT TRUE,
    auto_approve BOOLEAN DEFAULT FALSE,
    timeout_hours INT NULL,
    conditions JSON NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    
    INDEX idx_flow_id (flow_id),
    INDEX idx_step_order (step_order),
    INDEX idx_role_id (role_id),
    INDEX idx_branch_id (branch_id),
    INDEX idx_deleted_at (deleted_at),
    
    CONSTRAINT fk_flow_steps_flow
        FOREIGN KEY (flow_id)
        REFERENCES approval_flows(id)
        ON DELETE CASCADE,
    
    CONSTRAINT fk_flow_steps_role
        FOREIGN KEY (role_id)
        REFERENCES roles(id)
        ON DELETE SET NULL,
    
    CONSTRAINT fk_flow_steps_branch
        FOREIGN KEY (branch_id)
        REFERENCES branchs(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS approval_flow_steps;
-- +goose StatementEnd