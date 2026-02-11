-- +goose Up
-- Remove old fields (type, category)
-- +goose StatementBegin
ALTER TABLE approval_flows 
DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows 
DROP COLUMN category;
-- +goose StatementEnd

-- Add assignment system fields
-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN assignment_type ENUM('general','user_specific') DEFAULT 'general' AFTER approval_way;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN assigned_user_id CHAR(36) NULL AFTER assignment_type;
-- +goose StatementEnd

-- Add customization control fields
-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN is_customizable BOOLEAN DEFAULT FALSE AFTER assigned_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN allowed_creator_roles JSON NULL AFTER is_customizable;
-- +goose StatementEnd

-- Add custom flow tracking fields
-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN is_custom BOOLEAN DEFAULT FALSE AFTER allowed_creator_roles;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN created_by CHAR(36) NULL AFTER is_custom;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN base_flow_id CHAR(36) NULL AFTER created_by;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN custom_status ENUM('draft','pending_verification','approved','rejected') NULL AFTER base_flow_id;
-- +goose StatementEnd

-- Add verification fields
-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN verified_by CHAR(36) NULL AFTER custom_status;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN verified_at TIMESTAMP NULL AFTER verified_by;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN verification_notes TEXT NULL AFTER verified_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN rejection_reason TEXT NULL AFTER verification_notes;
-- +goose StatementEnd

-- Add foreign keys
-- +goose StatementBegin
ALTER TABLE approval_flows
ADD CONSTRAINT fk_flows_assigned_user 
    FOREIGN KEY (assigned_user_id) 
    REFERENCES users(id) 
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows
ADD CONSTRAINT fk_flows_created_by 
    FOREIGN KEY (created_by) 
    REFERENCES users(id) 
    ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows
ADD CONSTRAINT fk_flows_base_flow 
    FOREIGN KEY (base_flow_id) 
    REFERENCES approval_flows(id) 
    ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows
ADD CONSTRAINT fk_flows_verified_by 
    FOREIGN KEY (verified_by) 
    REFERENCES users(id) 
    ON DELETE SET NULL;
-- +goose StatementEnd

-- Add indexes
-- +goose StatementBegin
CREATE INDEX idx_assignment_type ON approval_flows(assignment_type);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_assigned_user_id ON approval_flows(assigned_user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_is_custom ON approval_flows(is_custom);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_created_by ON approval_flows(created_by);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_custom_status ON approval_flows(custom_status);
-- +goose StatementEnd

-- +goose Down
-- Drop indexes
-- +goose StatementBegin
DROP INDEX idx_custom_status ON approval_flows;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_created_by ON approval_flows;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_is_custom ON approval_flows;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_assigned_user_id ON approval_flows;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_assignment_type ON approval_flows;
-- +goose StatementEnd

-- Drop foreign keys
-- +goose StatementBegin
ALTER TABLE approval_flows DROP FOREIGN KEY fk_flows_verified_by;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP FOREIGN KEY fk_flows_base_flow;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP FOREIGN KEY fk_flows_created_by;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP FOREIGN KEY fk_flows_assigned_user;
-- +goose StatementEnd

-- Drop columns (reverse order)
-- +goose StatementBegin
ALTER TABLE approval_flows DROP COLUMN rejection_reason;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP COLUMN verification_notes;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP COLUMN verified_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP COLUMN verified_by;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP COLUMN custom_status;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP COLUMN base_flow_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP COLUMN created_by;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP COLUMN is_custom;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP COLUMN allowed_creator_roles;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP COLUMN is_customizable;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP COLUMN assigned_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP COLUMN assignment_type;
-- +goose StatementEnd

-- Restore old fields
-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN category ENUM('budget','non-budget','return','all') DEFAULT 'all';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows
ADD COLUMN type ENUM('it','non-it','all') DEFAULT 'all';
-- +goose StatementEnd