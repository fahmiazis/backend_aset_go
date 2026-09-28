package controllers

import (
	"backend-go/services"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// denyIfCannotViewTransaction — dipasang di endpoint detail transaksi.
// Menulis 403 dan mengembalikan true kalau user tidak punya akses ke cabang
// transaksi tersebut (lihat services.CanViewTransaction).
func denyIfCannotViewTransaction(c *gin.Context, transactionNumber, txType string) bool {
	if err := services.CanViewTransaction(assetViewer(c), transactionNumber, txType); err != nil {
		utils.ErrorResponse(c, http.StatusForbidden, err.Error())
		return true
	}
	return false
}
