package controllers

import (
	"backend-go/dto"
	"backend-go/services"
	"backend-go/utils"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// ============================================================================
// Upload master data dari Excel — branch, user, asset.
//   GET  /<entity>/import/template?mode=new|update → file template
//   POST /<entity>/import?mode=new|update&dry_run=true  (multipart "file")
// ============================================================================

const importMaxFileSize = 10 << 20 // 10 MB

func importMode(c *gin.Context) string {
	mode := c.Query("mode")
	if mode == "" {
		return dto.ImportModeNew
	}
	return mode
}

// openImportFile — nil kalau gagal (response sudah ditulis)
func openImportFile(c *gin.Context) io.ReadCloser {
	header, err := c.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "file is required")
		return nil
	}
	if strings.ToLower(filepath.Ext(header.Filename)) != ".xlsx" {
		utils.ErrorResponse(c, http.StatusBadRequest, "file must be an .xlsx workbook")
		return nil
	}
	if header.Size > importMaxFileSize {
		utils.ErrorResponse(c, http.StatusBadRequest, "file is larger than 10 MB")
		return nil
	}
	file, err := header.Open()
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "failed to open file")
		return nil
	}
	return file
}

func respondImport(c *gin.Context, run func(r io.Reader, mode string, dryRun bool) (*dto.ImportResult, error)) {
	file := openImportFile(c)
	if file == nil {
		return
	}
	defer file.Close()

	res, err := run(file, importMode(c), c.Query("dry_run") == "true")
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrImportFile) {
			status = http.StatusBadRequest
		}
		utils.ErrorResponse(c, status, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, services.ImportSummaryMessage(res), res)
}

func respondImportTemplate(c *gin.Context, entity string, build func(mode string) (*excelize.File, error)) {
	mode := importMode(c)
	file, err := build(mode)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrImportFile) {
			status = http.StatusBadRequest
		}
		utils.ErrorResponse(c, status, err.Error())
		return
	}
	writeReportExcel(c, file, "template-"+entity+"-"+mode+".xlsx")
}

// ---------- branch ----------

func ImportBranches(c *gin.Context) {
	respondImport(c, services.ImportBranches)
}

func BranchImportTemplate(c *gin.Context) {
	respondImportTemplate(c, "branch", services.BuildBranchImportTemplate)
}

// ---------- user ----------

func ImportUsers(c *gin.Context) {
	respondImport(c, services.ImportUsers)
}

func UserImportTemplate(c *gin.Context) {
	respondImportTemplate(c, "user", services.BuildUserImportTemplate)
}

// ---------- asset ----------

func ImportAssets(c *gin.Context) {
	viewer := assetViewer(c)
	respondImport(c, func(r io.Reader, mode string, dryRun bool) (*dto.ImportResult, error) {
		return services.ImportAssets(viewer, r, mode, dryRun)
	})
}

func AssetImportTemplate(c *gin.Context) {
	viewer := assetViewer(c)
	respondImportTemplate(c, "asset", func(mode string) (*excelize.File, error) {
		return services.BuildAssetImportTemplate(viewer, mode)
	})
}

// GET /assets/import/allowed — menyembunyikan tombol upload; sidebar tidak
// memuat menu bertipe permission
func CanImportAssets(c *gin.Context) {
	utils.SuccessResponse(c, http.StatusOK, "OK", gin.H{
		"allowed": services.CanImportAssets(c.GetString("user_id")),
	})
}

// ---------- download data (format template mass update) ----------

func respondImportExport(c *gin.Context, entity string, file *excelize.File, err error) {
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeReportExcel(c, file, "data-"+entity+"-"+time.Now().Format("20060102")+".xlsx")
}

// GET /branchs/export
func ExportBranches(c *gin.Context) {
	file, err := services.ExportBranches()
	respondImportExport(c, "branch", file, err)
}

// GET /users/export
func ExportUsers(c *gin.Context) {
	file, err := services.ExportUsers()
	respondImportExport(c, "user", file, err)
}

// GET /assets/export — filter sama dengan GET /assets (tanpa paging)
func ExportAssets(c *gin.Context) {
	var filter dto.AssetListFilter
	if v := c.Query("branch_code"); v != "" {
		filter.BranchCode = &v
	}
	if v := c.Query("asset_status"); v != "" {
		filter.AssetStatus = &v
	}
	if v := c.Query("search"); v != "" {
		filter.Search = &v
	}
	if v, err := strconv.ParseUint(c.Query("category_id"), 10, 64); err == nil {
		id := uint(v)
		filter.CategoryID = &id
	}
	file, err := services.ExportAssets(assetViewer(c), filter)
	respondImportExport(c, "asset", file, err)
}
