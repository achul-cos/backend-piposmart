package seeders

import (
	"backend_crm_piposmart/models"
	"backend_crm_piposmart/services"
	"fmt"
	"math/rand/v2"
	"strconv"

	"github.com/brianvoe/gofakeit/v7"
)

type CustomerSeeder struct {
	customerService *services.CustomerService
}

func NewCustomerSeeder() *CustomerSeeder {
	return &CustomerSeeder{
		customerService: services.NewCustomerService(),
	}
}

func (s *CustomerSeeder) Name() string {
	return "customer"
}

func (s *CustomerSeeder) Run(count int) error {
	// Mengetahui jumlah sales yang dimiliki
	sales, _ := services.NewSalesService().MenampilkanDataSales()
	jumlahSales := len(sales)

	for range count {

		salesId := uint64(rand.Int64N(int64(jumlahSales + 1)))

		customer := &models.Customer{
			KodeOwner:    strconv.Itoa(gofakeit.Number(10000, 99999)),
			NamaOwner:    gofakeit.Name(),
			NamaBrand:    gofakeit.Company(),
			NamaOutlet:   gofakeit.City(),
			KontakOwner:  gofakeit.Phone(),
			KontakOutlet: gofakeit.Phone(),
			SalesID:      &salesId,
		}

		if _, err := s.customerService.MenambahkanDataCustomer(customer); err != nil {
			return err
		}
	}

	fmt.Println("Berhasil seeding data customer")

	return nil
}
