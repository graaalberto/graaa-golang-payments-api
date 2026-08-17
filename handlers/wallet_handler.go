package handlers

import (
	"net/http"

	"frotago-payments-api/config"
	"frotago-payments-api/models"

	"github.com/gin-gonic/gin"
)

// GetDriverWallet retorna o saldo de ganhos e comissão do motorista
func GetDriverWallet(c *gin.Context) {
	driverID := c.Param("driverId")

	var wallet models.DriverWallet
	if err := config.DB.First(&wallet, "driver_id = ?", driverID).Error; err != nil {
		// Se não tiver registro, cria com saldo zero
		wallet = models.DriverWallet{
			DriverID:         driverID,
			AvailableBalance: 0.0,
			PendingBalance:   0.0,
			TotalEarnedAOA:   0.0,
			CompanyFeeAOA:    0.0,
		}
		config.DB.Create(&wallet)
	}

	c.JSON(http.StatusOK, wallet)
}
