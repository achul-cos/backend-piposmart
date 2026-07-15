package routes

import (
	"backend_crm_piposmart/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterAPIRoutes(router *gin.Engine) {

	customerController := controllers.NewCustomerController()
	router.GET("/customer", customerController.GetCustomers)
	router.POST("/customer", customerController.CreateCustomer)
	router.PATCH("/customer", customerController.UpdateCustomer)
	router.DELETE("/customer", customerController.DeleteCustomer)
	router.POST("/customer/restore", customerController.RestoreCustomer)
	router.DELETE("/customer/force", customerController.DeleteForceCustomer)

	salesController := controllers.NewSalesController()
	router.POST("/sales", salesController.CreateSales)
}
