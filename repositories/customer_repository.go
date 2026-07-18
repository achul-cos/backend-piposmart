package repositories

import (
	"backend_crm_piposmart/database"
	"backend_crm_piposmart/models"

	"gorm.io/gorm"
)

type CustomerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository() *CustomerRepository {
	return &CustomerRepository{
		db: database.DB,
	}
}

func (r *CustomerRepository) Create(customer *models.Customer) error {
	return r.db.Create(customer).Error
}

func (r *CustomerRepository) Read() ([]models.Customer, error) {
	customers := []models.Customer{}

	if err := r.db.Preload("Sales").Find(&customers).Error; err != nil {
		return nil, err
	}

	return customers, nil
}

func (r *CustomerRepository) ReadDeleted() ([]*models.Customer, error) {
	customers := []*models.Customer{}

	if err := r.db.Preload("Sales").Unscoped().Where("deleted_at IS NOT NULL").Find(&customers).Error; err != nil {
		return nil, err
	}

	return customers, nil
}

func (r *CustomerRepository) ReadAll() ([]*models.Customer, error) {
	customers := []*models.Customer{}

	if err := r.db.Preload("Sales").Unscoped().Find(&customers).Error; err != nil {
		return nil, err
	}

	return customers, nil
}

func (r *CustomerRepository) FindByID(id uint64) (*models.Customer, error) {
	customer := &models.Customer{}

	if err := r.db.First(customer, id).Error; err != nil {
		return nil, err
	}

	return customer, nil
}

func (r *CustomerRepository) Update(customer *models.Customer) (*models.Customer, error) {
	return customer, r.db.Save(customer).Error
}

func (r *CustomerRepository) Delete(id uint64) (*models.Customer, error) {
	error := r.db.Delete(&models.Customer{}, id).Error

	if error != nil {
		return nil, error
	}

	customer := &models.Customer{}

	if err := r.db.Unscoped().First(customer, id).Error; err != nil {
		return nil, err
	}

	return customer, nil
}

func (r *CustomerRepository) Restore(id uint64) (*models.Customer, error) {

	if err := r.db.Unscoped().Model(&models.Customer{}).Where("id = ?", id).Update("deleted_at", nil).Error; err != nil {
		return nil, err
	}

	customer, error := r.FindByID(id)

	if error != nil {
		return nil, error
	}

	return customer, nil
}

func (r *CustomerRepository) HardDelete(id uint64) (*models.Customer, error) {
	customer := &models.Customer{}

	if err := r.db.Unscoped().First(customer, id).Error; err != nil {
		return nil, err
	}

	return customer, r.db.Unscoped().Delete(&models.Customer{}, id).Error
}
