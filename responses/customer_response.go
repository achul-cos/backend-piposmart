package responses

import (
	"time"
)

type CustomerResponse struct {
	ID           uint64 `json:"id"`
	KodeOwner    string `json:"kode_owner"`
	NamaOwner    string `json:"nama_owner"`
	NamaBrand    string `json:"nama_brand"`
	NamaOutlet   string `json:"nama_outlet"`
	KontakOwner  string `json:"kontak_owner"`
	KontakOutlet string `json:"kontak_outler"`

	Sales *SalesSummaryResponse `json:"sales,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SalesSummaryResponse struct {
	ID        uint64 `json:"sales_id"`
	NameSales string `json:"sales_name"`
}
