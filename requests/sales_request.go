package requests

import (
	"backend_crm_piposmart/models"
	"backend_crm_piposmart/repositories"
)

type CreateSalesRequest struct {
	NamaSales     string `json:"nama_sales" binding:"required"`
	KontakSales   string `json:"kontak_sales" binding:"required"`
	EmailSales    string `json:"email_sales" binding:"required,email"`
	PasswordSales string `json:"password_sales" binding:"required"`
	IsSuperAdmin  bool   `json:"is_super_admin" gorm:"default:false"`
}

func (r *CreateSalesRequest) ToModel() *models.Sales {
	model := &models.Sales{
		NamaSales:     r.NamaSales,
		KontakSales:   r.KontakSales,
		EmailSales:    r.EmailSales,
		PasswordSales: r.PasswordSales,
		IsSuperAdmin:  r.IsSuperAdmin,
	}

	return model
}

type UpdateSalesRequest struct {
	ID            *uint64 `json:"id"`
	NamaSales     *string `json:"nama_sales"`
	KontakSales   *string `json:"kontak_sales"`
	EmailSales    *string `json:"email_sales"`
	PasswordSales *string `json:"password_sales"`
	IsSuperAdmin  *bool   `json:"is_super_admin"`
}

func (r *UpdateSalesRequest) ToModel() (*models.Sales, error) {
	sales, err := repositories.NewSalesRepository().FindById(*r.ID)

	if err != nil {
		return nil, err
	}

	model := &models.Sales{}

	model.ID = *r.ID
	model.CreatedAt = sales.CreatedAt

	if r.NamaSales == nil || *r.NamaSales == "" {
		model.NamaSales = sales.NamaSales
	} else {
		model.NamaSales = *r.NamaSales
	}

	if r.KontakSales == nil || *r.KontakSales == "" {
		model.KontakSales = sales.KontakSales
	} else {
		model.KontakSales = *r.KontakSales
	}

	if r.EmailSales == nil || *r.EmailSales == "" {
		model.EmailSales = sales.EmailSales
	} else {
		model.EmailSales = *r.EmailSales
	}

	if r.PasswordSales == nil || *r.PasswordSales == "" {
		model.PasswordSales = sales.PasswordSales
	} else {
		model.PasswordSales = *r.PasswordSales
	}

	if r.IsSuperAdmin == nil {
		model.IsSuperAdmin = sales.IsSuperAdmin
	} else {
		model.IsSuperAdmin = *r.IsSuperAdmin
	}

	return model, nil
}

type DeleteSalesRequest struct {
	ID uint64 `json:"id"`
}

func (r *DeleteSalesRequest) ToModel() *models.Sales {
	model := &models.Sales{}
	model.ID = r.ID

	return model
}

type RestoreSalesRequest struct {
	ID uint64 `json:"id"`
}

func (r *RestoreSalesRequest) ToModel() *models.Sales {
	model := &models.Sales{}
	model.ID = r.ID

	return model
}
