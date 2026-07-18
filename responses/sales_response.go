package responses

type SalesResponse struct {
	ID           uint64 `json:"id"`
	NamaSales    string `json:"nama_sales"`
	KontakSales  string `json:"kontak_sales"`
	EmailSales   string `json:"email_sales"`
	IsSuperAdmin bool   `json:"is_super_admin" gorm:"default:false"`
}
