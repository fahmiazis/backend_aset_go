package dto

import "time"

// WaitingNotification — satu pengajuan yang sedang menunggu tindakan user.
// Bukan notifikasi tersimpan: dihitung ulang dari aturan "Menunggu Saya"
// setiap kali diminta, jadi hilang sendiri begitu aksinya dikerjakan.
type WaitingNotification struct {
	TransactionType   string  `json:"transaction_type"` // procurement | mutation | disposal | disposal_agreement
	TransactionNumber string  `json:"transaction_number"`
	CurrentStage      string  `json:"current_stage"`
	CreatedByName     *string `json:"created_by_name"`
	// kapan pengajuan masuk ke stage/giliran saat ini
	Since time.Time `json:"since"`
}

type WaitingNotificationResponse struct {
	Total int                   `json:"total"`
	Items []WaitingNotification `json:"items"`
}
