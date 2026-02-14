package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// APPROVAL FLOW CONTROLLERS
// ============================================================================

// GetAllApprovalFlows - GET /approval-flows
func GetAllApprovalFlows(c *gin.Context) {
	flows, err := services.GetAllApprovalFlows()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval flows retrieved successfully", flows)
}

// GetApprovalFlowByID - GET /approval-flows/:id
func GetApprovalFlowByID(c *gin.Context) {
	id := c.Param("id")

	flow, err := services.GetApprovalFlowByID(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval flow retrieved successfully", flow)
}

// GetApprovalFlowByCode - GET /approval-flows/code/:code
func GetApprovalFlowByCode(c *gin.Context) {
	code := c.Param("code")

	flow, err := services.GetApprovalFlowByCode(code)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval flow retrieved successfully", flow)
}

// CreateApprovalFlow - POST /approval-flows
func CreateApprovalFlow(c *gin.Context) {
	var req dto.CreateApprovalFlowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	flow, err := services.CreateApprovalFlow(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Approval flow created successfully", flow)
}

// UpdateApprovalFlow - PUT /approval-flows/:id
func UpdateApprovalFlow(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateApprovalFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	flow, err := services.UpdateApprovalFlow(id, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval flow updated successfully", flow)
}

// DeleteApprovalFlow - DELETE /approval-flows/:id
func DeleteApprovalFlow(c *gin.Context) {
	id := c.Param("id")

	if err := services.DeleteApprovalFlow(id); err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval flow deleted successfully", nil)
}

// ============================================================================
// APPROVAL FLOW STEP CONTROLLERS
// ============================================================================

// CreateApprovalFlowStep - POST /approval-flow-steps
func CreateApprovalFlowStep(c *gin.Context) {
	var req dto.CreateApprovalFlowStepRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	step, err := services.CreateApprovalFlowStep(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Approval flow step created successfully", step)
}

// UpdateApprovalFlowStep - PUT /approval-flow-steps/:id
func UpdateApprovalFlowStep(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateApprovalFlowStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	step, err := services.UpdateApprovalFlowStep(id, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval flow step updated successfully", step)
}

// UpdateBulkStepOrderFlowStep - PUT /approval-flow-steps/:id
func UpdateBulkStepOrderFlowStep(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateBulkStepOrderFlowStep
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	err := services.UpdateBulkStepOrderFlowStep(id, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval flow step order updated successfully", nil)
}

// DeleteApprovalFlowStep - DELETE /approval-flow-steps/:id
func DeleteApprovalFlowStep(c *gin.Context) {
	id := c.Param("id")

	if err := services.DeleteApprovalFlowStep(id); err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Approval flow step deleted successfully", nil)
}

// ============================================================================
// TRANSACTION APPROVAL CONTROLLERS
// ============================================================================

// InitiateTransactionApproval - POST /transaction-approvals/initiate
func InitiateTransactionApproval(c *gin.Context) {
	var req dto.CreateTransactionApprovalRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.InitiateTransactionApproval(req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Transaction approval initiated successfully", nil)
}

// ApproveTransaction - POST /transaction-approvals/approve
func ApproveTransaction(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.ApproveTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.ApproveTransaction(userID, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Transaction approved successfully", nil)
}

// RejectTransaction - POST /transaction-approvals/reject
func RejectTransaction(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.RejectTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.RejectTransaction(userID, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Transaction rejected successfully", nil)
}

// GetTransactionApprovalStatus - GET /transaction-approvals/status/:transaction_number/:transaction_type
func GetTransactionApprovalStatus(c *gin.Context) {
	transactionNumber := c.Param("transaction_number")
	transactionType := c.Param("transaction_type")

	summary, err := services.GetTransactionApprovalStatus(transactionNumber, transactionType)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Transaction approval status retrieved successfully", summary)
}

// GetUserPendingApprovals - GET /transaction-approvals/pending
func GetUserPendingApprovals(c *gin.Context) {
	userID := c.GetString("user_id")

	approvals, err := services.GetUserPendingApprovals(userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Pending approvals retrieved successfully", approvals)
}
