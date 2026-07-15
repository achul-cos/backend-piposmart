package models

type Sales struct {
	BaseModel

	NamaSales     string `json:"nama_sales"`
	KontakSales   string `json:"kontak_sales"`
	EmailSales    string `json:"email_sales"`
	PasswordSales string `json:"password_sales"`
	IsSuperAdmin  bool   `json:"is_super_admin" gorm:"default:false"`
}
