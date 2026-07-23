package main

import (
	"backend_crm_piposmart/database"
	_ "backend_crm_piposmart/docs"
	"backend_crm_piposmart/routes"
	"backend_crm_piposmart/seeders"
	"log"
	"os"

	"github.com/brianvoe/gofakeit/v7"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// @title Backend CRM Piposmart
// @version 0.1
// @description Backend API untuk aplikasi frontend CRM Piposmart
// @BasePath /
func main() {

	// Koneksi ke database
	err := database.Connect()

	// Jika terdapat error saat koneksi ke database, maka hentikan fungsi main,
	// dan berikan pesan error
	if err != nil {
		log.Fatal(err)
	}

	args := os.Args

	switch args[1] {

	case "api":
		api()

	case "seed":
		gofakeit.Seed(0)

		switch args[2] {

		case "sales":
			seeders.NewSalesSeeder().Run()

		case "customer":
			seeders.NewCustomerSeeder().Run(100)

		default:
			seeders.NewSalesSeeder().Run()
			seeders.NewCustomerSeeder().Run(100)
		}
	default:
		api()

	}
}

func api() {
	// Selanjutnya daftarkan route
	router := gin.Default()
	router.Use(cors.Default())

	// Daftarkan Route Swagger
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "System Online",
		})
	})

	routes.RegisterAPIRoutes(router)

	// Jalankan router
	router.Run(":8080")
}
