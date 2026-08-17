package handlers

import (
	"fmt"
	"net/http"
	"time"

	"frotago-payments-api/config"
	"frotago-payments-api/models"
	"frotago-payments-api/services"

	"github.com/gin-gonic/gin"
)

// ChargeTrip inicia a cobrança da corrida via Multicaixa Express ou Dinheiro
func ChargeTrip(c *gin.Context) {
	var req struct {
		TripID        string  `json:"tripId" binding:"required"`
		AmountAOA     float64 `json:"amountAOA" binding:"required"`
		Method        string  `json:"method" binding:"required"` // multicaixa_express | cash
		CustomerPhone string  `json:"customerPhone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Parâmetros inválidos: " + err.Error()})
		return
	}

	paymentID := fmt.Sprintf("pay_%d", time.Now().UnixNano()%1000000)

	payment := models.Payment{
		ID:            paymentID,
		TripID:        req.TripID,
		AmountAOA:     req.AmountAOA,
		Method:        req.Method,
		Status:        "pending",
		CustomerPhone: req.CustomerPhone,
		CreatedAt:     time.Now(),
	}

	if req.Method == "multicaixa_express" {
		mcxResp, err := services.ProcessMulticaixaExpress(services.MulticaixaChargeRequest{
			PhoneNumber: req.CustomerPhone,
			AmountAOA:   req.AmountAOA,
			TripID:      req.TripID,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Falha na comunicação com Multicaixa Express"})
			return
		}
		payment.GatewayTransactionID = mcxResp.TransactionReference
	} else if req.Method == "cash" {
		// Pagamento em dinheiro físico ao motorista
		now := time.Now()
		payment.Status = "paid"
		payment.PaidAt = &now
		payment.GatewayTransactionID = "CASH_DIRECT"
	}

	if err := config.DB.Create(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Erro ao registrar pagamento: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Cobrança iniciada com sucesso",
		"payment": payment,
	})
}

// GetPaymentStatus consulta o status de um pagamento
func GetPaymentStatus(c *gin.Context) {
	id := c.Param("id")
	var payment models.Payment
	if err := config.DB.First(&payment, "id = ? OR trip_id = ?", id, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Pagamento não encontrado"})
		return
	}
	c.JSON(http.StatusOK, payment)
}
