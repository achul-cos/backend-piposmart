package models

import "backend_crm_piposmart/responses"

type Customer struct {
	BaseModel

	KodeOwner    string `json:"kode_owner"`
	NamaOwner    string `json:"nama_owner"`
	NamaBrand    string `json:"nama_brand"`
	NamaOutlet   string `json:"nama_outlet"`
	KontakOwner  string `json:"kontak_owner"`
	KontakOutlet string `json:"kontak_outler"`

	Sales   *Sales
	SalesID *uint64
}

func (c *Customer) ToReponse() *responses.CustomerResponse {
	response := &responses.CustomerResponse{
		ID:           c.ID,
		KodeOwner:    c.KodeOwner,
		NamaBrand:    c.NamaBrand,
		NamaOwner:    c.NamaOwner,
		KontakOwner:  c.KontakOwner,
		NamaOutlet:   c.NamaOutlet,
		KontakOutlet: c.KontakOutlet,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}

	if c.Sales != nil {
		response.Sales = &responses.SalesSummaryResponse{
			ID:        c.Sales.ID,
			NameSales: c.Sales.NamaSales,
		}
	}

	return response
}
