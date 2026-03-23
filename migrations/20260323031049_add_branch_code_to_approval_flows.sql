-- +goose Up

-- +goose StatementBegin
-- Drop existing unique index on flow_code
ALTER TABLE approval_flows DROP INDEX flow_code;
-- +goose StatementEnd

-- +goose StatementBegin
-- Add branch_code column
ALTER TABLE approval_flows
    ADD COLUMN branch_code VARCHAR(50) NOT NULL DEFAULT 'ALL'
        COMMENT 'Branch spesifik atau ALL untuk semua branch'
        AFTER flow_code;
-- +goose StatementEnd

-- +goose StatementBegin
-- Add new unique constraint on (flow_code, branch_code)
ALTER TABLE approval_flows
    ADD UNIQUE KEY uq_flow_code_branch (flow_code, branch_code);
-- +goose StatementEnd

-- +goose StatementBegin
-- Update existing data — semua existing flow jadi ALL
UPDATE approval_flows SET branch_code = 'ALL' WHERE branch_code = '';
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
ALTER TABLE approval_flows DROP INDEX uq_flow_code_branch;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows DROP COLUMN branch_code;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flows ADD UNIQUE KEY flow_code (flow_code);
-- +goose StatementEnd