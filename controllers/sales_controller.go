package controllers

import (
	"backend_crm_piposmart/requests"
	"backend_crm_piposmart/responses"
	"backend_crm_piposmart/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SalesController struct {
	salesService *services.SalesService
}

// fungsi untuk membuat objek SalesController
func NewSalesController() *SalesController {
	return &SalesController{
		salesService: services.NewSalesService(),
	}
}

// GetSales godoc
//
// @Summary			Menampilkan data sales
// @Descripttion	Menampilkan data sales dalam jumlah banyak atau semua
// @Tags			Sales
// @Accept			json
// @Produce			json
// @Success			200			{object}	responses.ApiResponse[[]responses.SalesResponse]
// @Failure			400			{object}	responses.ApiResponse[any]
// @Router			/sales	[get]
func (c *SalesController) GetSales(ctx *gin.Context) {
	// Nanti kalau ada param ada disini
	// ...

	sales, err := c.salesService.MenampilkanDataSales()

	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[any]{
			Message: err.Error(),
			Data:    nil,
		})

		return
	}

	if sales != nil {
		response := []responses.SalesResponse{}
		for _, sale := range sales {
			response = append(response, *sale.ToResponse())
		}

		ctx.JSON(http.StatusOK, responses.ApiResponse[[]responses.SalesResponse]{
			Message: "Berikut Data Sales",
			Data:    response,
		})
	}
}

// CreateSales godoc
//
// @Summary			Membuat data sales
// @Description		Menambahkan data sales kedalam tabel sales atau halaman kelola sales
// @Tags			Sales
// @Accept			json
// @Produce			json
// @Param			request body		requests.CreateSalesRequest true "Create Data Sales"
// @Success			201		{object}	responses.ApiResponse[responses.SalesResponse]
// @Failure			400		{object}	responses.ApiResponse[requests.CreateSalesRequest]
// @Router			/sales 	[post]
func (c *SalesController) CreateSales(ctx *gin.Context) {
	request := requests.CreateSalesRequest{}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.CreateSalesRequest]{
			Message: err.Error(),
			Data:    request,
		})

		return
	}

	sales := request.ToModel()

	if _, err := c.salesService.MenambahkanDataSales(sales); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.CreateSalesRequest]{
			Message: err.Error(),
			Data:    request,
		})

		return
	}

	// Buat Response
	response := sales.ToResponse()

	// Berikan Response
	ctx.JSON(http.StatusCreated, responses.ApiResponse[responses.SalesResponse]{
		Message: "Data Sales Berhasil dibuat",
		Data:    *response,
	})
}

// UpdateSales godoc
//
// @Summary			Mengubah data sales secara patch
// @Description		Mengubah data sales sebagaian atau semua data sales
// @Tags			Sales
// @Accept			json
// @Produce			json
// @Param			request		body		requests.UpdateSalesRequest		true		"Update Data Sales"
// @Success			202			{object}	responses.ApiResponse[responses.SalesResponse]
// @Failure			400			{object}	responses.ApiResponse[requests.UpdateSalesRequest]
// @Router			/sales	[patch]
func (c *SalesController) UpdateSales(ctx *gin.Context) {
	// Mengambil request
	request := requests.UpdateSalesRequest{}

	// Binding
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.UpdateSalesRequest]{
			Message: err.Error(),
			Data:    request,
		})

		return
	}

	// Mengubah request menjadi model
	sales, error := request.ToModel()

	if error != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.UpdateSalesRequest]{
			Message: error.Error(),
			Data:    request,
		})

		return
	}

	// Jalankan Service
	salesUpdated, errorUpdate := c.salesService.MengubahDataSales(sales)

	if errorUpdate != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.UpdateSalesRequest]{
			Message: errorUpdate.Error(),
			Data:    request,
		})

		return
	}

	// Berikan response
	response := salesUpdated.ToResponse()

	ctx.JSON(http.StatusCreated, responses.ApiResponse[responses.SalesResponse]{
		Message: "Data Sales Berhasil diubah",
		Data:    *response,
	})
}

// DeleteSales godoc
//
// @Summary			Menghapus (soft delete) data sales
// @Description		Menghapus data sales tetapi dapat dipulihkan nantinya
// @Tags			Sales
// @Accept			json
// @Produce			json
// @Param			request		body		requests.DeleteSalesRequest 	true		"Delete Data Sales"
// @Success			202			{object}	responses.ApiResponse[responses.SalesResponse]
// @Failure			400			{object}	responses.ApiResponse[requests.DeleteSalesRequest]
// @Router			/sales		[delete]
func (c *SalesController) DeleteSales(ctx *gin.Context) {
	request := requests.DeleteSalesRequest{}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.DeleteSalesRequest]{
			Message: err.Error(),
			Data:    request,
		})

		return
	}

	sales := request.ToModel()

	// Jalankan Service
	salesDeleted, errorDelete := c.salesService.MenghapusDataSales(sales)

	if errorDelete != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.DeleteSalesRequest]{
			Message: errorDelete.Error(),
			Data:    request,
		})

		return
	}

	response := salesDeleted.ToResponse()

	ctx.JSON(http.StatusAccepted, responses.ApiResponse[responses.SalesResponse]{
		Message: "Data Sales Berhasil Dihapus, Dapat Dipulihkan di tong sampah.",
		Data:    *response,
	})
}

// RestoreSales godoc
//
// @Summary			Memulihkan data sales
// @Description		Memulihkan data sales yang terhapus (soft delete)
// @Tags			Sales
// @Accept			json
// @Produce			json
// @Param			request		body		requests.RestoreSalesRequest 	true		"Restore Data Sales"
// @Success			202			{object}	responses.ApiResponse[responses.SalesResponse]
// @Failure			400			{object}	responses.ApiResponse[requests.RestoreSalesRequest]
// @Router			/sales/restore		[post]
func (c *SalesController) RestoreSales(ctx *gin.Context) {
	request := requests.RestoreSalesRequest{}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.RestoreSalesRequest]{
			Message: err.Error(),
			Data:    request,
		})

		return
	}

	// Mengubah menjadi model
	sales := request.ToModel()

	// Jalankan service
	salesRestored, errorRestore := c.salesService.MemulihkanDataSales(sales)

	if errorRestore != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.RestoreSalesRequest]{
			Message: errorRestore.Error(),
			Data:    request,
		})

		return
	}

	response := salesRestored.ToResponse()

	ctx.JSON(http.StatusAccepted, responses.ApiResponse[responses.SalesResponse]{
		Message: "Data Sales Telah di pulihkan",
		Data:    *response,
	})
}

// DeleteForceSales godoc
//
// @Summary			Menghapus secara permanen data sales
// @Description		Menghapus secara permanen data sales (hard delete)
// @Tags			Sales
// @Accept			json
// @Produce			json
// @Param			request		body		requests.DeleteSalesRequest 	true		"Hard Delete Data Sales"
// @Success			202			{object}	responses.ApiResponse[responses.SalesResponse]
// @Failure			400			{object}	responses.ApiResponse[requests.DeleteSalesRequest]
// @Router			/sales/force		[delete]
func (c *SalesController) DeleteForceSales(ctx *gin.Context) {
	request := requests.DeleteSalesRequest{}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.DeleteSalesRequest]{
			Message: err.Error(),
			Data:    request,
		})

		return
	}

	// Mengubah menjadi model
	sales := request.ToModel()

	// Jalankan service
	salesDeletedForce, errorDelete := c.salesService.MenghapusDataSalesPermanen(sales)

	if errorDelete != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[requests.DeleteSalesRequest]{
			Message: errorDelete.Error(),
			Data:    request,
		})

		return
	}

	// Buat response
	response := salesDeletedForce.ToResponse()

	ctx.JSON(http.StatusAccepted, responses.ApiResponse[responses.SalesResponse]{
		Message: "Data Sales Telah dihapus permanen",
		Data:    *response,
	})
}

// GetSalesDeleted godoc
//
// @Summary			Menampilkan data-data sales yang terhapus
// @Description		Menampilkan data-data sales yang terhapus (Soft delete)
// @Tags			Sales
// @Accept			json
// @Produce			json
// @Success			200			{object}	responses.ApiResponse[[]responses.SalesResponse]
// @Failure			400			{object}	responses.ApiResponse[any]
// @Router			/sales/deleted	[get]
func (c *SalesController) GetSalesDeleted(ctx *gin.Context) {
	// Kalau ada param, bakal akan di koding disini
	// ...

	// Jalankan service
	salesDeleted, err := c.salesService.MenampilkanDataSalesTerhapus()

	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[any]{
			Message: err.Error(),
			Data:    nil,
		})

		return
	}

	if salesDeleted != nil {
		response := []*responses.SalesResponse{}
		for _, sale := range salesDeleted {
			response = append(response, sale.ToResponse())
		}

		ctx.JSON(http.StatusOK, responses.ApiResponse[[]*responses.SalesResponse]{
			Message: "Berikut data customer yang telah terhapus",
			Data:    response,
		})
	}
}

// GetSalesAll godoc
//
// @Summary			Menampilkan semua data-data sales bahkan yang telah terhapus
// @Description		Menampilkan semua data-data sales bahkan yang telah terhapus
// @Tags			Sales
// @Accept			json
// @Produce			json
// @Success			200			{object}	responses.ApiResponse[[]responses.SalesResponse]
// @Failure			400			{object}	responses.ApiResponse[any]
// @Router			/sales/all	[get]
func (c *SalesController) GetSalesAll(ctx *gin.Context) {
	// Kalau ada param, bakal akan dikoding disini
	// ...

	// Jalankan service
	salesAll, err := c.salesService.MenampilkanSalesSemua()

	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ApiResponse[any]{
			Message: err.Error(),
			Data:    nil,
		})

		return
	}

	if salesAll != nil {
		response := []*responses.SalesResponse{}
		for _, sale := range salesAll {
			response = append(response, sale.ToResponse())
		}

		ctx.JSON(http.StatusOK, responses.ApiResponse[[]*responses.SalesResponse]{
			Message: "Berikut semua data-data sales termasuk yang terhapus",
			Data:    response,
		})
	}
}
