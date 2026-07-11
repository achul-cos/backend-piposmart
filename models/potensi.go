package models

import "time"

type Potensi struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	TanggalInput     string    `gorm:"type:varchar(10);column:tanggal_input" json:"tanggalInput"`
	SumberData       string    `gorm:"type:varchar(50);column:sumber_data" json:"sumberData"`
	PicNasabah       string    `gorm:"type:varchar(100);column:pic_nasabah" json:"picNasabah"`
	KodeOwner        string    `gorm:"type:varchar(50);column:kode_owner" json:"kodeOwner"`
	NamaOwner        string    `gorm:"type:varchar(100);column:nama_owner" json:"namaOwner"`
	Brand            string    `gorm:"type:varchar(100);column:brand" json:"brand"`
	HpOwner          string    `gorm:"type:varchar(30);column:hp_owner" json:"hpOwner"`
	Status           string    `gorm:"type:varchar(50);column:status" json:"status"`
	Membership       string    `gorm:"type:varchar(50);column:membership" json:"membership"`
	TargetPaket      string    `gorm:"type:varchar(100);column:target_paket" json:"targetPaket"`
	TargetNominal    float64   `gorm:"type:decimal(15,2);column:target_nominal" json:"targetNominal"`
	PaketAktual      string    `gorm:"type:varchar(100);column:paket_aktual" json:"paketAktual"`
	NominalAktual    float64   `gorm:"type:decimal(15,2);column:nominal_aktual" json:"nominalAktual"`
	TanggalTraining  string    `gorm:"type:varchar(10);column:tanggal_training" json:"tanggalTraining"`
	StatusTraining   string    `gorm:"type:varchar(100);column:status_training" json:"statusTraining"`
	TimelineTraining string    `gorm:"type:text;column:timeline_training" json:"timelineTraining"`
	RatingPlaystore  string    `gorm:"type:varchar(100);column:rating_playstore" json:"ratingPlaystore"`
	// 🌟 TAMBAHAN PARAMETER VALIDASI CLOSING FRONTEND
	TanggalRealisasi string    `gorm:"type:varchar(10);column:tanggal_realisasi" json:"tanggalRealisasi"`
	BuktiTransfer    string    `gorm:"type:text;column:bukti_transfer" json:"buktiTransfer"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}