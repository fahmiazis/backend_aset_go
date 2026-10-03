package dto

// ============================================================================
// UPLOAD MASTER DATA (branch, user, asset) dari file Excel
//
// Satu file diproses utuh: kalau ada satu baris saja yang gagal validasi,
// tidak ada yang disimpan. dry_run=true hanya memvalidasi (preview).
// ============================================================================

const (
	ImportModeNew    = "new"    // baris baru, kode dibentuk otomatis
	ImportModeUpdate = "update" // mass update data yang sudah ada
	ImportModeMember = "member" // branch saja: set homebase & akses cabang user
)

// ImportRowError — satu kesalahan pada satu baris. Code dipakai frontend untuk
// menerjemahkan pesan (i18n masterImport.errors.<code>); Message cadangan
// berbahasa Inggris kalau kodenya belum dikenal frontend.
type ImportRowError struct {
	Code    string            `json:"code"`
	Field   string            `json:"field,omitempty"`
	Params  map[string]string `json:"params,omitempty"`
	Message string            `json:"message"`
}

type ImportRowResult struct {
	// nomor baris di Excel (header = baris 1)
	Row    int               `json:"row"`
	Values map[string]string `json:"values"`
	Errors []ImportRowError  `json:"errors"`
	// kode yang dibentuk / diubah: branch_code, username, asset_number.
	// Saat dry run mode new, branch_code berisi perkiraan.
	Result string `json:"result,omitempty"`
}

type ImportResult struct {
	Entity    string            `json:"entity"`
	Mode      string            `json:"mode"`
	DryRun    bool              `json:"dry_run"`
	Columns   []string          `json:"columns"`
	TotalRows int               `json:"total_rows"`
	ValidRows int               `json:"valid_rows"`
	ErrorRows int               `json:"error_rows"`
	Imported  int               `json:"imported"`
	Rows      []ImportRowResult `json:"rows"`
}
