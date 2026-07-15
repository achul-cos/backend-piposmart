package requests

type CreateSalesRequest struct {
	NamaSales     string `json:"nama_sales" binding:"required"`
	KontakSales   string `json:"kontak_sales" binding:"required"`
	EmailSales    string `json:"email_sales" binding:"required,email"`
	PasswordSales string `json:"password_sales" binding:"required"`
	IsSuperAdmin  bool   `json:"is_super_admin" gorm:"default:false"`
}
