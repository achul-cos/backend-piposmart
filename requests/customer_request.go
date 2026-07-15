package requests

import (
	"backend_crm_piposmart/models"
	"backend_crm_piposmart/repositories"
)

type CreateCustomerRequest struct {
	KodeOwner    string  `json:"kode_owner" binding:"required"`
	NamaOwner    string  `json:"nama_owner" binding:"required"`
	NamaBrand    string  `json:"nama_brand" binding:"required"`
	NamaOutlet   string  `json:"nama_outlet" binding:"required"`
	KontakOwner  string  `json:"kontak_owner" binding:"required"`
	KontakOutlet string  `json:"kontak_outlet" binding:"required"`
	SalesID      *uint64 `json:"sales_id,omitempty"`
}

func (r *CreateCustomerRequest) ToModel() *models.Customer {

	model := &models.Customer{
		KodeOwner:    r.KodeOwner,
		NamaOwner:    r.NamaOwner,
		NamaBrand:    r.NamaBrand,
		NamaOutlet:   r.NamaOutlet,
		KontakOwner:  r.KontakOwner,
		KontakOutlet: r.KontakOutlet,
	}

	if *r.SalesID <= 0 {
		model.SalesID = nil
		model.Sales = nil
	}

	return model
}

type UpdateCustomerRequest struct {
	ID           *uint64 `json:"id"`
	KodeOwner    *string `json:"kode_owner"`
	NamaOwner    *string `json:"nama_owner"`
	NamaBrand    *string `json:"nama_brand"`
	NamaOutlet   *string `json:"nama_outlet"`
	KontakOwner  *string `json:"kontak_owner"`
	KontakOutlet *string `json:"kontak_outlet"`
	SalesID      *uint64 `json:"sales_id"`
}

func (r *UpdateCustomerRequest) ToModel() (*models.Customer, error) {
	customer, err := repositories.NewCustomerRepository().FindByID(*r.ID)

	if err != nil {
		return nil, err
	}

	model := &models.Customer{}

	model.ID = *r.ID
	model.CreatedAt = customer.CreatedAt

	if r.KodeOwner == nil || *r.KodeOwner == "" {
		model.KodeOwner = customer.KodeOwner
	} else {
		model.KodeOwner = *r.KodeOwner
	}

	if r.NamaOwner == nil || *r.NamaOwner == "" {
		model.NamaOwner = customer.NamaOwner
	} else {
		model.NamaOwner = *r.NamaOwner
	}

	if r.NamaBrand == nil || *r.NamaBrand == "" {
		model.NamaBrand = customer.NamaBrand
	} else {
		model.NamaBrand = *r.NamaBrand
	}

	if r.NamaOutlet == nil || *r.NamaOutlet == "" {
		model.NamaOutlet = customer.NamaOutlet
	} else {
		model.NamaOutlet = *r.NamaOutlet
	}

	if r.KontakOwner == nil || *r.KontakOwner == "" {
		model.KontakOwner = customer.KontakOwner
	} else {
		model.KontakOwner = *r.KontakOwner
	}

	if r.KontakOutlet == nil || *r.KontakOutlet == "" {
		model.KontakOutlet = customer.KontakOutlet
	} else {
		model.KontakOutlet = *r.KontakOutlet
	}

	if r.SalesID == nil || *r.SalesID == 0 {
		model.SalesID = customer.SalesID
	} else {
		if _, err := repositories.NewSalesRepository().FindById(*r.SalesID); err != nil {
			model.SalesID = nil
		} else {
			model.SalesID = r.SalesID
		}
	}

	return model, nil
}

type DeleteCustomerRequest struct {
	ID uint64 `json:"id"`
}

func (r *DeleteCustomerRequest) ToModel() *models.Customer {
	model := &models.Customer{}
	model.ID = r.ID

	return model
}

type RestoreCustomerRequest struct {
	ID uint64 `json:"id"`
}

func (r *RestoreCustomerRequest) ToModel() *models.Customer {
	model := &models.Customer{}
	model.ID = r.ID

	return model
}
