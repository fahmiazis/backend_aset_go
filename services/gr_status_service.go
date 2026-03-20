package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
)

func GetProcurementGRStatusDetail(transactionNumber string) (*dto.AssetGRStatusResponse, error) {
	// Get transaction
	transaction, err := getProcurementTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	if transaction.CurrentStage != models.StageGR &&
		transaction.CurrentStage != models.StageFinished {
		return nil, errors.New("transaction is not in GR or SELESAI stage")
	}

	// Get procurement items yang sudah diverifikasi sebagai ASSET
	var verifications []models.TransactionItemVerification
	config.DB.
		Preload("TransactionProcurement").
		Where("transaction_id = ? AND item_type = ? AND is_active = ?",
			transaction.ID, models.ItemTypeAsset, true).
		Find(&verifications)

	// Get semua GR records untuk transaksi ini
	var grRecords []models.AssetGR
	config.DB.Where("transaction_id = ?", transaction.ID).Find(&grRecords)
	grMap := make(map[uint]models.AssetGR) // key: asset_id
	for _, gr := range grRecords {
		grMap[gr.AssetID] = gr
	}

	totalAssets := 0
	grDone := 0
	items := make([]dto.GRItemResponse, 0)

	for _, verif := range verifications {
		if verif.TransactionProcurement == nil {
			continue
		}
		proc := verif.TransactionProcurement

		// Get semua asset untuk procurement item ini
		var acquisitions []models.AssetAcquisition
		config.DB.
			Preload("Asset").
			Where("transaction_procurement_id = ?", proc.ID).
			Find(&acquisitions)

		assetDetails := make([]dto.AssetGRDetail, 0, len(acquisitions))
		for _, acq := range acquisitions {
			if acq.Asset == nil || acq.AssetID == nil {
				continue
			}

			totalAssets++
			grStatus := "PENDING_RECEIPT"
			var grDate *string
			var grBy *string

			if gr, hasGR := grMap[*acq.AssetID]; hasGR {
				grDone++
				grStatus = "AVAILABLE"
				dateStr := gr.GRDate.Format("2006-01-02")
				grDate = &dateStr
				grBy = &gr.GRBy
			}

			assetDetails = append(assetDetails, dto.AssetGRDetail{
				AssetID:     acq.Asset.ID,
				AssetNumber: acq.Asset.AssetNumber,
				AssetName:   acq.Asset.AssetName,
				BranchCode:  acq.BranchCode,
				IONumber:    acq.IONumber,
				GRStatus:    grStatus,
				GRDate:      grDate,
				GRBy:        grBy,
			})
		}

		items = append(items, dto.GRItemResponse{
			ProcurementItemID: proc.ID,
			ItemName:          proc.ItemName,
			Quantity:          proc.Quantity,
			BranchCode:        proc.BranchCode,
			Assets:            assetDetails,
		})
	}

	return &dto.AssetGRStatusResponse{
		TransactionNumber: transactionNumber,
		CurrentStage:      transaction.CurrentStage,
		TotalAssets:       totalAssets,
		GRDone:            grDone,
		GRPending:         totalAssets - grDone,
		Items:             items,
	}, nil
}
