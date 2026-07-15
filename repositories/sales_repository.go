package repositories

import (
	"backend_crm_piposmart/database"
	"backend_crm_piposmart/models"

	"gorm.io/gorm"
)

type SalesRepository struct {
	db *gorm.DB
}

func NewSalesRepository() *SalesRepository {
	return &SalesRepository{
		db: database.DB,
	}
}

func (r *SalesRepository) FindById(id uint64) (*models.Sales, error) {

	sales := &models.Sales{}

	if err := r.db.First(sales, &id).Error; err != nil {
		return nil, err
	}

	return sales, nil
}
