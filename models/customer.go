package models

import (
	"backend_crm_piposmart/responses"
)

type Customer struct {
	BaseModel

	KodeOwner    string `json:"kode_owner"`
	NamaOwner    string `json:"nama_owner"`
	NamaBrand    string `json:"nama_brand"`
	NamaOutlet   string `json:"nama_outlet"`
	KontakOwner  string `json:"kontak_owner"`
	KontakOutlet string `json:"kontak_outlet"`

	Sales   *Sales
	SalesID *uint64

	CustomerStatus *CustomerStatus `gorm:"constraint:OnDelete:CASCADE;"`

	CallHistories     []CallHistory     `gorm:"constraint:OnDelete:CASCADE;" json:"call_histories"`
	TrainingHistories []TrainingHistory `gorm:"constraint:OnDelete:CASCADE;" json:"training_histories"`
	PurchaseHistories []PurchaseHistory `gorm:"constraint:OnDelete:CASCADE;" json:"purchase_histories"`
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

		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}

	if c.CustomerStatus != nil {
		response.CallStatus = c.CustomerStatus.CallStatus
		response.ChatStatus = c.CustomerStatus.ChatStatus
		response.TanggalFu = c.CustomerStatus.TanggalFu
		response.TotalFu = c.CustomerStatus.TotalFu
		response.Noted = c.CustomerStatus.Noted
		response.Remarks = c.CustomerStatus.Remarks
		response.Score = c.CustomerStatus.Score
		response.StatusAkun = c.CustomerStatus.StatusAkun
		response.FinalisasiClosing = c.CustomerStatus.FinalisasiClosing
		response.Nominal = c.CustomerStatus.Nominal
	}

	if c.Sales != nil {
		response.Sales = &responses.SalesSummaryResponse{
			ID:        c.Sales.ID,
			NameSales: c.Sales.NamaSales,
		}
	}

	// Map CallHistories
	response.CallHistories = make([]responses.CallHistoryResponse, 0)
	for _, history := range c.CallHistories {
		response.CallHistories = append(response.CallHistories, responses.CallHistoryResponse{
			WaktuCall:  history.WaktuCall,
			PicSales:   history.PicSales,
			Remark:     history.Remark,
			Conclusion: history.Conclusion,
		})
	}

	// Map TrainingHistories
	response.TrainingHistories = make([]responses.TrainingHistoryResponse, 0)
	for _, history := range c.TrainingHistories {
		response.TrainingHistories = append(response.TrainingHistories, responses.TrainingHistoryResponse{
			WaktuTraining:  history.WaktuTraining,
			LokasiTraining: history.LokasiTraining,
		})
	}

	// Map PurchaseHistories
	response.PurchaseHistories = make([]responses.PurchaseHistoryResponse, 0)
	for _, history := range c.PurchaseHistories {
		response.PurchaseHistories = append(response.PurchaseHistories, responses.PurchaseHistoryResponse{
			Paket:         history.Paket,
			WaktuMulai:    history.WaktuMulai,
			WaktuBerakhir: history.WaktuBerakhir,
			HargaAktual:   history.HargaAktual,
		})
	}

	return response
}
