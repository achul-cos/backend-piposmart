package controllers

import (
	"backend_crm_piposmart/requests"
	"backend_crm_piposmart/responses"
	"backend_crm_piposmart/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CustomerController struct {
	customerService *services.CustomerService
}

// Membuat objek CustomerController pada api.go
func NewCustomerController() *CustomerController {
	return &CustomerController{
		customerService: services.NewCustomerService(),
	}
}

// GetCustomers godoc
//
// @Summary			Menampilkan data semua customer
// @Descripttion	Menampilkan data dalam jumlah yang banyak, biasanya untuk tabel
// @Tags			Customer
// @Accept			json
// @Produce			json
// @Success			200			{object}	responses.ApiResponse[[]responses.CustomerResponse]
// @Failure			400			{object}	responses.ApiResponse[any]
// @Router			/customer	[get]
func (c *CustomerController) GetCustomers(ctx *gin.Context) {
	customers, err := c.customerService.MenampilkanDataCustomer()

	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[any]{
			Message: err.Error(),
			Data:    nil,
		})
	}

	if customers != nil {
		response := []responses.CustomerResponse{}
		for _, customer := range customers {
			response = append(response, *customer.ToReponse())
		}

		ctx.JSON(http.StatusOK, responses.ApiResponse[[]responses.CustomerResponse]{
			Message: "Berikut Data Customer",
			Data:    response,
		})
	}
}

// CreateCustomer godoc
//
// @Summary			Membuat data customer
// @Descripttion	Menambahkan data customer kedalam database
// @Tags			Customer
// @Accept			json
// @Produce			json
// @Param			request		body		requests.CreateCustomerRequest		true		"Create Data Customer"
// @Success			201			{object}	responses.ApiResponse[responses.CustomerResponse]
// @Failure			400			{object}	responses.ApiResponse[requests.CreateCustomerRequest]
// @Router			/customer	[post]
func (c *CustomerController) CreateCustomer(ctx *gin.Context) {

	// Deklarasikan sebuah data request
	request := requests.CreateCustomerRequest{}

	// Lakukan binding gin.Context dengan request
	// Jika user mengirimkan data tidak sesuai request, berikan pesan error
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.CreateCustomerRequest]{
			Message: err.Error(),
			Data:    request,
		})

		return
	}

	// Ubah request menjadi model
	customer := request.ToModel()

	// Jalan fungsi service untuk membuat customer
	if _, err := c.customerService.MenambahkanDataCustomer(customer); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.CreateCustomerRequest]{
			Message: err.Error(),
			Data:    request,
		})

		return
	}

	// Buat response
	response := customer.ToReponse()

	// Berikan response
	ctx.JSON(http.StatusCreated, responses.ApiResponse[responses.CustomerResponse]{
		Message: "Customer berhasil dibuat",
		Data:    *response,
	})
}

// UpdateCustomer godoc
//
// @Summary			Mengubah Data Customer
// @Descripttion	Mengubah Data Customer
// @Tags			Customer
// @Accept			json
// @Produce			json
// @Param			request		body		requests.UpdateCustomerRequest		true		"Update Data Customer"
// @Success			201			{object}	responses.ApiResponse[responses.CustomerResponse]
// @Failure			400			{object}	responses.ApiResponse[requests.UpdateCustomerRequest]
// @Router			/customer	[patch]
func (c *CustomerController) UpdateCustomer(ctx *gin.Context) {
	// Mengambil request
	request := requests.UpdateCustomerRequest{}

	// Binding
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.UpdateCustomerRequest]{
			Message: err.Error(),
			Data:    request,
		})

		return
	}

	// Mengubah menjadi model
	customer, error := request.ToModel()

	if error != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.UpdateCustomerRequest]{
			Message: error.Error(),
			Data:    request,
		})
	}

	// Jalankan service
	if _, err := c.customerService.MengubahDataCustomer(customer); err != nil {
		ctx.JSON(http.StatusInternalServerError, responses.ApiResponse[requests.UpdateCustomerRequest]{
			Message: err.Error(),
			Data:    request,
		})

		return
	}

	// Berikan Reponse
	response := customer.ToReponse()

	ctx.JSON(http.StatusCreated, responses.ApiResponse[responses.CustomerResponse]{
		Message: "Data berhasil diubah",
		Data:    *response,
	})
}

// DeleteCustomer godoc
//
// @Summary			Menghapus (soft delete) data customer
// @Descripttion	Menghapus data customer tetapi dapa dipulihkan nantinya
// @Tags			Customer
// @Accept			json
// @Produce			json
// @Param			request		body		requests.DeleteCustomerRequest		true		"Delete Data Customer"
// @Success			202			{object}	responses.ApiResponse[responses.CustomerResponse]
// @Failure			400			{object}	responses.ApiResponse[requests.DeleteCustomerRequest]
// @Router			/customer	[delete]
func (c *CustomerController) DeleteCustomer(ctx *gin.Context) {
	request := requests.DeleteCustomerRequest{}

	// Binding
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.DeleteCustomerRequest]{
			Message: err.Error(),
			Data:    request,
		})
	}

	// Mengubah menjadi model
	customer := request.ToModel()

	// Jalankan service
	customerDeleted, err := c.customerService.MenghapusDataCustomer(customer)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.DeleteCustomerRequest]{
			Message: err.Error(),
			Data:    request,
		})
	}

	// Berikan Responses
	response := customerDeleted.ToReponse()

	ctx.JSON(http.StatusAccepted, responses.ApiResponse[responses.CustomerResponse]{
		Message: "Data Berhasil Terhapus",
		Data:    *response,
	})
}

// RestoreCustomer godoc
//
// @Summary			Memulihkan Data Customer
// @Descripttion	Memulihakan data customer yang terhapus (soft delete)
// @Tags			Customer
// @Accept			json
// @Produce			json
// @Param			request		body		requests.RestoreCustomerRequest 	true		"Restore Data Customer"
// @Success			202			{object}	responses.ApiResponse[responses.CustomerResponse]
// @Failure			400			{object}	responses.ApiResponse[requests.RestoreCustomerRequest]
// @Router			/customer/restore		[post]
func (c *CustomerController) RestoreCustomer(ctx *gin.Context) {
	request := requests.RestoreCustomerRequest{}

	// binding
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.RestoreCustomerRequest]{
			Message: err.Error(),
			Data:    request,
		})
	}

	// Mengubah menjadi model
	customer := request.ToModel()

	// Jalankan Service
	customerServiced, err := c.customerService.MemulihkanDataCustomer(customer)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.RestoreCustomerRequest]{
			Message: err.Error(),
			Data:    request,
		})
	}

	// Berikan Response
	response := customerServiced.ToReponse()

	ctx.JSON(http.StatusAccepted, responses.ApiResponse[responses.CustomerResponse]{
		Message: "Data Customer Telah Dipulihakan",
		Data:    *response,
	})
}

// DeleteForceCustomer godoc
//
// @Summary			Menghapus secara permanen data customer
// @Descripttion	Menghapus secara permanen data customer (hard delete)
// @Tags			Customer
// @Accept			json
// @Produce			json
// @Param			request		body		requests.DeleteCustomerRequest 	true		"Hard Delete Data Customer"
// @Success			202			{object}	responses.ApiResponse[responses.CustomerResponse]
// @Failure			400			{object}	responses.ApiResponse[requests.DeleteCustomerRequest]
// @Router			/customer/force		[delete]
func (c *CustomerController) DeleteForceCustomer(ctx *gin.Context) {
	request := requests.DeleteCustomerRequest{}

	// Binding
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.DeleteCustomerRequest]{
			Message: err.Error(),
			Data:    request,
		})
	}

	// Mengubah menjadi model
	customer := request.ToModel()

	// Jalankan service
	customerDeletedForce, err := c.customerService.MenghapusPermanenDataCustomer(customer)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.DeleteCustomerRequest]{
			Message: err.Error(),
			Data:    request,
		})
	}

	// Berikan response
	response := customerDeletedForce.ToReponse()

	ctx.JSON(http.StatusAccepted, responses.ApiResponse[responses.CustomerResponse]{
		Message: "Data customer berhasil di hapus permanen",
		Data:    *response,
	})
}
