package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// CUSTOM APPROVAL CRUD
// ============================================================================

// CreateCustomApproval - POST /custom-approvals
func CreateCustomApproval(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.CreateCustomApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.CreateCustomApproval(userID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Custom approval created and pending verification", result)
}

// GetUserCustomApprovals - GET /custom-approvals/me
func GetUserCustomApprovals(c *gin.Context) {
	userID := c.GetString("user_id")

	results, err := services.GetUserCustomApprovals(userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Custom approvals retrieved successfully", results)
}

// GetCustomApprovalByID - GET /custom-approvals/:id
// func GetCustomApprovalByID(c *gin.Context) {
// 	id := c.Param("id")

// 	result, err := services.GetCustomApprovalByID(id)
// 	if err != nil {
// 		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
// 		return
// 	}

// 	utils.SuccessResponse(c, http.StatusOK, "Custom approval retrieved successfully", result)
// }

// UpdateCustomApproval - PUT /custom-approvals/:id
func UpdateCustomApproval(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	var req dto.UpdateCustomApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	result, err := services.UpdateCustomApproval(userID, id, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Custom approval updated and pending verification", result)
}

// DeleteCustomApproval - DELETE /custom-approvals/:id
func DeleteCustomApproval(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	if err := services.DeleteCustomApproval(userID, id); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Custom approval deleted successfully", nil)
}

// ============================================================================
// VERIFICATION (Tim Asset / Admin)
// ============================================================================

// GetPendingVerifications - GET /custom-approvals/pending-verifications
func GetPendingVerifications(c *gin.Context) {
	results, err := services.GetPendingVerifications()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Pending verifications retrieved successfully", results)
}

// VerifyCustomApproval - POST /custom-approvals/:id/verify
func VerifyCustomApproval(c *gin.Context) {
	verifierID := c.GetString("user_id")
	id := c.Param("id")

	var req dto.VerifyCustomApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := services.VerifyCustomApproval(verifierID, id, req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	action := "approved"
	if req.Action == "reject" {
		action = "rejected"
	}

	utils.SuccessResponse(c, http.StatusOK, "Custom approval "+action+" successfully", nil)
}

// ============================================================================
// HELPER - Check if user can customize flow
// ============================================================================

// CheckCanCustomizeFlow - GET /approval-flows/:id/can-customize
func CheckCanCustomizeFlow(c *gin.Context) {
	userID := c.GetString("user_id")
	flowID := c.Param("id")

	canCustomize, err := services.CanUserCustomizeFlow(userID, flowID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]bool{
		"can_customize": canCustomize,
	}

	utils.SuccessResponse(c, http.StatusOK, "Permission check completed", response)
}
