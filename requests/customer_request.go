package requests

import (
	"backend_crm_piposmart/models"
	"backend_crm_piposmart/repositories"
	"time"
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

	if r.SalesID == nil || *r.SalesID <= 0 {
		model.SalesID = nil
		model.Sales = nil
	} else {
		model.SalesID = r.SalesID
	}

	return model
}

type CallHistoryRequest struct {
	WaktuCall  string `json:"waktu_call"`
	PicSales   string `json:"pic_sales"`
	Remark     string `json:"remark"`
	Conclusion string `json:"conclusion"`
}

type TrainingHistoryRequest struct {
	WaktuTraining  string `json:"waktu_training"`
	LokasiTraining string `json:"lokasi_training"`
}

type PurchaseHistoryRequest struct {
	Paket         string  `json:"paket"`
	WaktuMulai    string  `json:"waktu_mulai"`
	WaktuBerakhir string  `json:"waktu_berakhir"`
	HargaAktual   float64 `json:"harga_aktual"`
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

	CallStatus        *string  `json:"call_status"`
	ChatStatus        *string  `json:"chat_status"`
	TanggalFu         *string  `json:"tanggal_fu"`
	TotalFu           *int     `json:"total_fu"`
	Noted             *string  `json:"noted"`
	Remarks           *string  `json:"remarks"`
	Score             *int     `json:"score"`
	StatusAkun        *string  `json:"status_akun"`
	FinalisasiClosing *string  `json:"finalisasi_closing"`
	Nominal           *float64 `json:"nominal"`

	CallHistories     *[]CallHistoryRequest     `json:"call_histories"`
	TrainingHistories *[]TrainingHistoryRequest `json:"training_histories"`
	PurchaseHistories *[]PurchaseHistoryRequest `json:"purchase_histories"`
}

func parseTime(timeStr *string) *time.Time {
	if timeStr == nil || *timeStr == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *timeStr)
	if err != nil {
		// Fallback for simple date or local datetime
		t, err = time.Parse("2006-01-02T15:04", *timeStr)
		if err != nil {
			t, err = time.Parse("2006-01-02", *timeStr)
			if err != nil {
				return nil
			}
		}
	}
	return &t
}

func parseTimeRaw(timeStr string) *time.Time {
	return parseTime(&timeStr)
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

	if customer.CustomerStatus != nil {
		model.CustomerStatus = &models.CustomerStatus{
			BaseModel:         customer.CustomerStatus.BaseModel,
			CustomerID:        customer.CustomerStatus.CustomerID,
			CallStatus:        customer.CustomerStatus.CallStatus,
			ChatStatus:        customer.CustomerStatus.ChatStatus,
			TanggalFu:         customer.CustomerStatus.TanggalFu,
			TotalFu:           customer.CustomerStatus.TotalFu,
			Noted:             customer.CustomerStatus.Noted,
			Remarks:           customer.CustomerStatus.Remarks,
			Score:             customer.CustomerStatus.Score,
			StatusAkun:        customer.CustomerStatus.StatusAkun,
			FinalisasiClosing: customer.CustomerStatus.FinalisasiClosing,
			Nominal:           customer.CustomerStatus.Nominal,
		}
	} else {
		model.CustomerStatus = &models.CustomerStatus{
			CustomerID: model.ID,
		}
	}

	if r.CallStatus != nil { model.CustomerStatus.CallStatus = r.CallStatus }
	if r.ChatStatus != nil { model.CustomerStatus.ChatStatus = r.ChatStatus }
	if r.TanggalFu != nil { model.CustomerStatus.TanggalFu = parseTime(r.TanggalFu) }
	if r.TotalFu != nil { model.CustomerStatus.TotalFu = r.TotalFu }
	if r.Noted != nil { model.CustomerStatus.Noted = r.Noted }
	if r.Remarks != nil { model.CustomerStatus.Remarks = r.Remarks }
	if r.Score != nil { model.CustomerStatus.Score = r.Score }
	if r.StatusAkun != nil { model.CustomerStatus.StatusAkun = r.StatusAkun }
	if r.FinalisasiClosing != nil { model.CustomerStatus.FinalisasiClosing = r.FinalisasiClosing }
	if r.Nominal != nil { model.CustomerStatus.Nominal = r.Nominal }

	if r.CallHistories != nil {
		for _, h := range *r.CallHistories {
			model.CallHistories = append(model.CallHistories, models.CallHistory{
				CustomerID: model.ID,
				WaktuCall:  parseTimeRaw(h.WaktuCall),
				PicSales:   h.PicSales,
				Remark:     h.Remark,
				Conclusion: h.Conclusion,
			})
		}
	}

	if r.TrainingHistories != nil {
		for _, h := range *r.TrainingHistories {
			model.TrainingHistories = append(model.TrainingHistories, models.TrainingHistory{
				CustomerID:     model.ID,
				WaktuTraining:  parseTimeRaw(h.WaktuTraining),
				LokasiTraining: h.LokasiTraining,
			})
		}
	}

	if r.PurchaseHistories != nil {
		for _, h := range *r.PurchaseHistories {
			model.PurchaseHistories = append(model.PurchaseHistories, models.PurchaseHistory{
				CustomerID:    model.ID,
				Paket:         h.Paket,
				WaktuMulai:    parseTimeRaw(h.WaktuMulai),
				WaktuBerakhir: parseTimeRaw(h.WaktuBerakhir),
				HargaAktual:   h.HargaAktual,
			})
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
