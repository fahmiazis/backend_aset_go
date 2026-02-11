-- +goose Up
-- +goose StatementBegin
SET @pr_flow_id = UUID();
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO approval_flows (id, flow_code, flow_name, type, category, approval_way, description, is_active)
VALUES (
    @pr_flow_id,
    'PR_APPROVAL',
    'Purchase Request Approval',
    'all',
    'budget',
    'sequential',
    'Standard approval flow for purchase requests',
    TRUE
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO approval_flow_steps (id, flow_id, step_order, step_name, step_role, is_required, is_visible)
VALUES 
    (UUID(), @pr_flow_id, 1, 'Creator', 'creator', TRUE, TRUE),
    (UUID(), @pr_flow_id, 2, 'Department Manager', 'approver', TRUE, TRUE),
    (UUID(), @pr_flow_id, 3, 'Finance Review', 'approver', TRUE, TRUE),
    (UUID(), @pr_flow_id, 4, 'Director Approval', 'approver', TRUE, FALSE);
-- +goose StatementEnd

-- +goose StatementBegin
SET @transfer_flow_id = UUID();
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO approval_flows (id, flow_code, flow_name, type, category, approval_way, description, is_active)
VALUES (
    @transfer_flow_id,
    'TRANSFER_APPROVAL',
    'Employee Transfer Approval',
    'all',
    'all',
    'sequential',
    'Approval flow for employee transfers between branches',
    TRUE
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO approval_flow_steps (id, flow_id, step_order, step_name, step_role, structure, is_required, is_visible)
VALUES 
    (UUID(), @transfer_flow_id, 1, 'HR Creator', 'creator', NULL, TRUE, TRUE),
    (UUID(), @transfer_flow_id, 2, 'Sender Branch Manager', 'approver', 'sender_manager', TRUE, TRUE),
    (UUID(), @transfer_flow_id, 3, 'Receiver Branch Manager', 'receiver', 'receiver_manager', TRUE, TRUE),
    (UUID(), @transfer_flow_id, 4, 'HR Finalization', 'approver', NULL, TRUE, TRUE);
-- +goose StatementEnd

-- +goose StatementBegin
SET @it_flow_id = UUID();
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO approval_flows (id, flow_code, flow_name, type, category, approval_way, description, is_active)
VALUES (
    @it_flow_id,
    'IT_REQUEST_APPROVAL',
    'IT Equipment Request Approval',
    'it',
    'non-budget',
    'sequential',
    'Approval flow for IT equipment requests',
    TRUE
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO approval_flow_steps (id, flow_id, step_order, step_name, step_role, is_required, is_visible)
VALUES 
    (UUID(), @it_flow_id, 1, 'Requestor', 'creator', TRUE, TRUE),
    (UUID(), @it_flow_id, 2, 'IT Manager Review', 'reviewer', TRUE, TRUE),
    (UUID(), @it_flow_id, 3, 'IT Director Approval', 'approver', TRUE, TRUE);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM approval_flow_steps WHERE flow_id IN (
    SELECT id FROM approval_flows WHERE flow_code IN ('PR_APPROVAL', 'TRANSFER_APPROVAL', 'IT_REQUEST_APPROVAL')
);
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM approval_flows WHERE flow_code IN ('PR_APPROVAL', 'TRANSFER_APPROVAL', 'IT_REQUEST_APPROVAL');
-- +goose StatementEnd