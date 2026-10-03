package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetAllAssets(c *gin.Context) {
	var filter dto.AssetListFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 10
	}

	assets, total, err := services.GetAllAssets(filter, assetViewer(c))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"data":  assets,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	}

	utils.SuccessResponse(c, http.StatusOK, "Assets retrieved successfully", response)
}

func assetViewer(c *gin.Context) services.AssetViewer {
	return services.AssetViewer{UserID: c.GetString("user_id"), IsAdmin: isAdminRequest(c)}
}

// GET /assets/my-branches — cabang yang boleh dilihat user (dropdown filter)
func GetViewableAssetBranches(c *gin.Context) {
	branches, err := services.GetViewableAssetBranches(assetViewer(c))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Branches retrieved successfully", branches)
}

func GetAssetByNumber(c *gin.Context) {
	assetNumber := c.Param("number")

	asset, err := services.GetAssetByNumber(assetNumber)
	// aset cabang lain dibalas "tidak ditemukan", bukan 403 — tidak
	// membocorkan bahwa nomor aset itu ada
	if err != nil || !services.CanViewAsset(assetViewer(c), asset) {
		utils.ErrorResponse(c, http.StatusNotFound, "asset not found")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset retrieved successfully", asset)
}

func GetAssetValueHistory(c *gin.Context) {
	assetNumber := c.Param("number")

	if asset, err := services.GetAssetByNumber(assetNumber); err != nil || !services.CanViewAsset(assetViewer(c), asset) {
		utils.ErrorResponse(c, http.StatusNotFound, "asset not found")
		return
	}

	history, err := services.GetAssetValueHistory(assetNumber)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset value history retrieved successfully", history)
}

// POST /assets/qr-codes — isi QR (nomor aset terenkripsi) untuk label aset.
// Dibatasi cabang user seperti GET /assets.
func GetAssetQRCodes(c *gin.Context) {
	var req dto.AssetQRCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	codes, err := services.GetAssetQRCodes(req.AssetNumbers, assetViewer(c))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrQRKeyMissing) {
			status = http.StatusServiceUnavailable
		}
		utils.ErrorResponse(c, status, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset QR codes generated successfully", codes)
}

// POST /assets/qr/resolve — hasil scan label → detail aset. Hanya aplikasi
// (lewat backend) yang bisa membuka isi QR; aset cabang lain dibalas 404.
func ResolveAssetQR(c *gin.Context) {
	var req dto.AssetQRResolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	assetNumber, err := services.DecryptAssetQR(req.Payload)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrQRKeyMissing) {
			status = http.StatusServiceUnavailable
		}
		utils.ErrorResponse(c, status, err.Error())
		return
	}

	asset, err := services.GetAssetByNumber(assetNumber)
	if err != nil || !services.CanViewAsset(assetViewer(c), asset) {
		utils.ErrorResponse(c, http.StatusNotFound, "asset not found")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Asset retrieved successfully", asset)
}
