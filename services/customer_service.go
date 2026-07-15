package services

import (
	"backend_crm_piposmart/models"
	"backend_crm_piposmart/repositories"
)

type CustomerService struct {
	customerRepository *repositories.CustomerRepository
	salesRepository    *repositories.SalesRepository
}

func NewCustomerService() *CustomerService {
	return &CustomerService{
		customerRepository: repositories.NewCustomerRepository(),
		salesRepository:    repositories.NewSalesRepository(),
	}
}

func (s *CustomerService) MenambahkanDataCustomer(customer *models.Customer) (*models.Customer, error) {

	// Cek apakah sales_id yang dimasukkan oleh pengguna itu
	// terdaftar pada tabel sales. Jika tidak maka kosongin Sales ID nya
	if customer.SalesID != nil {
		if _, err := s.salesRepository.FindById(*customer.SalesID); err != nil {
			customer.SalesID = nil
		}
	}

	// Manambahkan user
	if err := s.customerRepository.Create(customer); err != nil {
		return nil, err
	}

	return customer, nil
}

func (s *CustomerService) MenampilkanDataCustomer() ([]models.Customer, error) {

	customers, err := s.customerRepository.Read()

	if err != nil {
		return nil, err
	}

	return customers, nil
}

func (s *CustomerService) MengubahDataCustomer(customer *models.Customer) (*models.Customer, error) {
	customerUpdated, err := s.customerRepository.Update(customer)

	if err != nil {
		return nil, err
	}

	return customerUpdated, nil
}

func (s *CustomerService) MenghapusDataCustomer(customer *models.Customer) (*models.Customer, error) {

	customerDeleted, err := s.customerRepository.Delete(customer.ID)

	if err != nil {
		return nil, err
	}

	return customerDeleted, nil
}

func (s *CustomerService) MemulihkanDataCustomer(customer *models.Customer) (*models.Customer, error) {
	customerRestored, err := s.customerRepository.Restore(customer.ID)

	if err != nil {
		return nil, err
	}

	return customerRestored, nil
}

func (s *CustomerService) MenghapusPermanenDataCustomer(customer *models.Customer) (*models.Customer, error) {
	customerDeletedForce, err := s.customerRepository.HardDelete(customer.ID)

	if err != nil {
		return nil, err
	}

	return customerDeletedForce, nil
}
