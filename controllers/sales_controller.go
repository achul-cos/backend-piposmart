package controllers

import (
	"backend_crm_piposmart/requests"
	"backend_crm_piposmart/responses"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SalesController struct {
}

// fungsi untuk membuat objek SalesController
func NewSalesController() *SalesController {
	return &SalesController{}
}

// CreateSales godoc
//
// @Summary			Membuat data sales
// @Description		Menambahkan data sales kedalam tabel sales atau halaman kelola sales
// @Tags			Sales
// @Accept			json
// @Produce			json
// @Param			request body		requests.CreateSalesRequest true "Data Sales"
// @Success			201		{object}	responses.ApiResponse[responses.SalesResponse]
// @Failure			400		{object}	responses.ApiResponse[requests.CreateSalesRequest]
// @Router			/sales 	[POST]
func (sc *SalesController) CreateSales(c *gin.Context) {

	// Schema Request
	request := requests.CreateSalesRequest{}

	// Lakukan binding request dengan schema
	err := c.ShouldBindJSON(&request)

	// Jika ada error, maka kembalikan pesan error dan hentikan fungsi
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ApiResponse[requests.CreateSalesRequest]{
			Message: err.Error(),
			Data:    request,
		})

		return
	}

	// Selanjutnya jalankan fungsi service
	// ...

	// Selanjutnya buatlah reponse berdasarkan schema
	response := responses.SalesResponse{
		ID:          100,
		NamaSales:   request.NamaSales,
		KontakSales: request.KontakSales,
		EmailSales:  request.EmailSales,
	}

	// Kirimkan reponse mengguanakan json
	c.JSON(http.StatusCreated, responses.ApiResponse[responses.SalesResponse]{
		Message: "Data Berhasil Ditambahkan",
		Data:    response,
	})
}
