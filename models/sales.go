package models

import "backend_crm_piposmart/responses"

type Sales struct {
	BaseModel

	NamaSales     string `json:"nama_sales"`
	KontakSales   string `json:"kontak_sales"`
	EmailSales    string `json:"email_sales"`
	PasswordSales string `json:"password_sales"`
	IsSuperAdmin  bool   `json:"is_super_admin" gorm:"default:false"`
}

func (s *Sales) ToResponse() *responses.SalesResponse {
	response := &responses.SalesResponse{
		ID:           s.ID,
		NamaSales:    s.NamaSales,
		KontakSales:  s.KontakSales,
		EmailSales:   s.EmailSales,
		IsSuperAdmin: s.IsSuperAdmin,
	}
	return response
}
