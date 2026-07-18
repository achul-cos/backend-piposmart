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

func (r *SalesRepository) Create(sales *models.Sales) (*models.Sales, error) {

	if err := r.db.Create(sales).Error; err != nil {
		return nil, err
	}

	return sales, nil
}

func (r *SalesRepository) Read() ([]*models.Sales, error) {
	sales := []*models.Sales{}

	if err := r.db.Find(&sales).Error; err != nil {
		return nil, err
	}

	return sales, nil
}

func (r *SalesRepository) ReadDeleted() ([]*models.Sales, error) {
	sales := []*models.Sales{}

	if err := r.db.Unscoped().Where("deleted_at IS NOT NULL").Find(&sales).Error; err != nil {
		return nil, err
	}

	return sales, nil
}

func (r *SalesRepository) ReadAll() ([]*models.Sales, error) {
	sales := []*models.Sales{}

	if err := r.db.Unscoped().Find(&sales).Error; err != nil {
		return nil, err
	}

	return sales, nil
}

func (r *SalesRepository) FindById(id uint64) (*models.Sales, error) {
	sales := &models.Sales{}

	if err := r.db.First(sales, &id).Error; err != nil {
		return nil, err
	}

	return sales, nil
}

func (r *SalesRepository) Update(sales *models.Sales) (*models.Sales, error) {
	if err := r.db.Save(sales).Error; err != nil {
		return nil, err
	}

	return sales, nil
}

func (r *SalesRepository) Delete(id uint64) (*models.Sales, error) {
	if err := r.db.Delete(&models.Sales{}, id).Error; err != nil {
		return nil, err
	}

	sales := &models.Sales{}

	if err := r.db.Unscoped().First(sales, id).Error; err != nil {
		return nil, err
	}

	return sales, nil
}

func (r *SalesRepository) Restore(id uint64) (*models.Sales, error) {
	if err := r.db.Unscoped().Model(&models.Sales{}).Where("id = ?", id).Update("deleted_at", nil).Error; err != nil {
		return nil, err
	}

	sales, error := r.FindById(id)

	if error != nil {
		return nil, error
	}

	return sales, nil
}

func (r *SalesRepository) HardDelete(id uint64) (*models.Sales, error) {
	sales := &models.Sales{}

	if err := r.db.Unscoped().First(sales, id).Error; err != nil {
		return nil, err
	}

	if err := r.db.Unscoped().Delete(&models.Sales{}, id).Error; err != nil {
		return nil, err
	}

	return sales, nil
}
