package handlers

import (
	"errors"
	"net/http"

	"frotago-payments-api/config"
	"frotago-payments-api/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetDriverWallet retorna o saldo de ganhos e comissão do motorista
func GetDriverWallet(c *gin.Context) {
	driverID := c.Param("driverId")
	if c.GetString("userRole") != "admin" {
		userID := c.GetString("userID")
		var driver struct {
			UserID *string
		}
		if userID == "" || config.DB.Table("drivers").Select("user_id").Where("id = ?", driverID).Take(&driver).Error != nil ||
			driver.UserID == nil || *driver.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acesso negado"})
			return
		}
	}

	var wallet models.DriverWallet
	if err := config.DB.First(&wallet, "driver_id = ?", driverID).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Não foi possível consultar a carteira"})
			return
		}
		wallet = models.DriverWallet{
			DriverID:         driverID,
			AvailableBalance: 0.0,
			PendingBalance:   0.0,
			TotalEarnedAOA:   0.0,
			CompanyFeeAOA:    0.0,
		}
		if err := config.DB.Create(&wallet).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Não foi possível criar a carteira"})
			return
		}
	}

	c.JSON(http.StatusOK, wallet)
}
