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
	router.GET("/customer/all", customerController.GetCustomersAll)
	router.GET("/customer/deleted", customerController.GetCustomersDeleted)

	salesController := controllers.NewSalesController()
	router.GET("/sales", salesController.GetSales)
	router.POST("sales", salesController.CreateSales)
	router.PATCH("/sales", salesController.UpdateSales)
	router.DELETE("/sales", salesController.DeleteSales)
	router.POST("/sales/restore", salesController.RestoreSales)
	router.DELETE("/sales/force", salesController.DeleteForceSales)
	router.GET("/sales/deleted", salesController.GetSalesDeleted)
	router.GET("/sales/all", salesController.GetSalesAll)
}
