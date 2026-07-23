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
	KontakOutlet string `json:"kontak_outlet"`

	Sales *SalesSummaryResponse `json:"sales,omitempty"`

	CallStatus        *string    `json:"call_status"`
	ChatStatus        *string    `json:"chat_status"`
	TanggalFu         *time.Time `json:"tanggal_fu"`
	TotalFu           *int       `json:"total_fu"`
	Noted             *string    `json:"noted"`
	Remarks           *string    `json:"remarks"`
	Score             *int       `json:"score"`
	StatusAkun        *string    `json:"status_akun"`
	FinalisasiClosing *string    `json:"finalisasi_closing"`
	Nominal           *float64   `json:"nominal"`

	CallHistories     []CallHistoryResponse     `json:"call_histories"`
	TrainingHistories []TrainingHistoryResponse `json:"training_histories"`
	PurchaseHistories []PurchaseHistoryResponse `json:"purchase_histories"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SalesSummaryResponse struct {
	ID        uint64 `json:"sales_id"`
	NameSales string `json:"sales_name"`
}

type CallHistoryResponse struct {
	WaktuCall  *time.Time `json:"waktu_call"`
	PicSales   string     `json:"pic_sales"`
	Remark     string     `json:"remark"`
	Conclusion string     `json:"conclusion"`
}

type TrainingHistoryResponse struct {
	WaktuTraining  *time.Time `json:"waktu_training"`
	LokasiTraining string     `json:"lokasi_training"`
}

type PurchaseHistoryResponse struct {
	Paket         string     `json:"paket"`
	WaktuMulai    *time.Time `json:"waktu_mulai"`
	WaktuBerakhir *time.Time `json:"waktu_berakhir"`
	HargaAktual   float64    `json:"harga_aktual"`
}
