package services

import (
	"backend_crm_piposmart/models"
	"backend_crm_piposmart/repositories"
)

type SalesService struct {
	salesRepository *repositories.SalesRepository
}

func NewSalesService() *SalesService {
	return &SalesService{
		salesRepository: repositories.NewSalesRepository(),
	}
}

func (s *SalesService) MenambahkanDataSales(sales *models.Sales) (*models.Sales, error) {
	if _, err := s.salesRepository.Create(sales); err != nil {
		return nil, err
	}
	return sales, nil
}

func (s *SalesService) MenampilkanDataSales() ([]*models.Sales, error) {
	sales, err := s.salesRepository.Read()

	if err != nil {
		return nil, err
	}

	return sales, nil
}

func (s *SalesService) MengubahDataSales(sales *models.Sales) (*models.Sales, error) {
	salesUpdated, err := s.salesRepository.Update(sales)

	if err != nil {
		return nil, err
	}

	return salesUpdated, nil
}

func (s *SalesService) MenghapusDataSales(sales *models.Sales) (*models.Sales, error) {
	salesDeleted, err := s.salesRepository.Delete(sales.ID)

	if err != nil {
		return nil, err
	}

	return salesDeleted, nil
}

func (s *SalesService) MemulihkanDataSales(sales *models.Sales) (*models.Sales, error) {
	salesRestored, err := s.salesRepository.Restore(sales.ID)

	if err != nil {
		return nil, err
	}

	return salesRestored, nil
}

func (s *SalesService) MenghapusDataSalesPermanen(sales *models.Sales) (*models.Sales, error) {
	salesDeletedForce, err := s.salesRepository.HardDelete(sales.ID)

	if err != nil {
		return nil, err
	}

	return salesDeletedForce, nil
}

func (s *SalesService) MenampilkanDataSalesTerhapus() ([]*models.Sales, error) {
	salesDeleted, err := s.salesRepository.ReadDeleted()

	if err != nil {
		return nil, err
	}

	return salesDeleted, nil
}

func (s *SalesService) MenampilkanSalesSemua() ([]*models.Sales, error) {
	salesAll, err := s.salesRepository.ReadAll()

	if err != nil {
		return nil, err
	}

	return salesAll, nil
}
