package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"frotago-payments-api/config"
	"frotago-payments-api/models"

	"github.com/gin-gonic/gin"
)

// MulticaixaWebhook recebe o callback de confirmação da EMIS
func MulticaixaWebhook(c *gin.Context) {
	var payload struct {
		TransactionReference string  `json:"transactionReference" binding:"required"`
		Status               string  `json:"status" binding:"required"` // PAID, FAILED, TIMEOUT
		AmountAOA            float64 `json:"amountAOA"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Payload inválido"})
		return
	}

	var payment models.Payment
	if err := config.DB.First(&payment, "gateway_transaction_id = ?", payload.TransactionReference).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Transação não encontrada"})
		return
	}

	if payload.Status == "PAID" {
		now := time.Now()
		payment.Status = "paid"
		payment.PaidAt = &now
		config.DB.Save(&payment)

		// Notifica a API Central da Frota (Fleet Core) para concluir a corrida
		fleetURL := os.Getenv("FLEET_API_URL")
		if fleetURL == "" {
			fleetURL = "http://localhost:8082"
		}

		go func(tripID string) {
			updateURL := fmt.Sprintf("%s/api/trips/%s/status", fleetURL, tripID)
			body, _ := json.Marshal(map[string]interface{}{
				"status": "completed",
			})
			req, _ := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: 5 * time.Second}
			_, _ = client.Do(req)
		}(payment.TripID)
	} else {
		payment.Status = "failed"
		config.DB.Save(&payment)
	}

	c.JSON(http.StatusOK, gin.H{"status": "received", "paymentStatus": payment.Status})
}
