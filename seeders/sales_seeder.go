package seeders

import (
	"backend_crm_piposmart/database"
	"backend_crm_piposmart/models"
	"fmt"
)

type SalesSeeder struct {
}

func NewSalesSeeder() *SalesSeeder {
	return &SalesSeeder{}
}

func (s *SalesSeeder) Name() string {
	return "sales"
}

func (s *SalesSeeder) Run() error {

	sales := []models.Sales{
		{
			NamaSales:     "Achul",
			KontakSales:   "08111111111",
			EmailSales:    "achul@piposmart.com",
			PasswordSales: "achul123",
			IsSuperAdmin:  false,
		},
		{
			NamaSales:     "Satria",
			KontakSales:   "08222222222",
			EmailSales:    "satria@piposmart.com",
			PasswordSales: "satria123",
			IsSuperAdmin:  false,
		},
		{
			NamaSales:     "Lidya",
			KontakSales:   "08333333333",
			EmailSales:    "lidya@piposmart.com",
			PasswordSales: "lidya123",
			IsSuperAdmin:  false,
		},
		{
			NamaSales:     "Wati",
			KontakSales:   "08444444444",
			EmailSales:    "wati@piposmart.com",
			PasswordSales: "wati123",
			IsSuperAdmin:  true,
		},
	}

	for _, sale := range sales {
		database.DB.Where(&models.Sales{
			EmailSales: sale.EmailSales,
		}).FirstOrCreate(&sale)
	}

	fmt.Println("Berhasil melakukan seeder di data sales.")

	return nil
}
