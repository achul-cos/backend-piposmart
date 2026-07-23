package models

import "time"

type CallHistory struct {
	BaseModel
	CustomerID uint64     `json:"customer_id"`
	WaktuCall  *time.Time `json:"waktu_call"`
	PicSales   string     `json:"pic_sales"`
	Remark     string     `json:"remark"`
	Conclusion string     `json:"conclusion"`
}

type TrainingHistory struct {
	BaseModel
	CustomerID     uint64     `json:"customer_id"`
	WaktuTraining  *time.Time `json:"waktu_training"`
	LokasiTraining string     `json:"lokasi_training"`
}

type PurchaseHistory struct {
	BaseModel
	CustomerID    uint64     `json:"customer_id"`
	Paket         string     `json:"paket"`
	WaktuMulai    *time.Time `json:"waktu_mulai"`
	WaktuBerakhir *time.Time `json:"waktu_berakhir"`
	HargaAktual   float64    `json:"harga_aktual"`
}
