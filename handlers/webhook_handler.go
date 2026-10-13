package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"frotago-payments-api/config"
	"frotago-payments-api/models"
	"frotago-payments-api/services"

	"github.com/gin-gonic/gin"
)

const maxWebhookBodyBytes = 1 << 20

func MulticaixaWebhook(c *gin.Context) {
	secret := os.Getenv("MULTICAIXA_WEBHOOK_SECRET")
	if len(secret) < 32 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Webhook não configurado"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookBodyBytes+1))
	if err != nil || len(body) > maxWebhookBodyBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"status": "error", "message": "Payload inválido"})
		return
	}
	if !validWebhookSignature(body, c.GetHeader("X-Multicaixa-Signature"), secret) {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Assinatura do webhook inválida"})
		return
	}

	var payload struct {
		TransactionReference string  `json:"transactionReference" binding:"required"`
		Status               string  `json:"status" binding:"required"`
		AmountAOA            float64 `json:"amountAOA" binding:"required"`
	}
	if err := json.Unmarshal(body, &payload); err != nil ||
		(payload.Status != "PAID" && payload.Status != "FAILED" && payload.Status != "TIMEOUT") ||
		payload.AmountAOA <= 0 || math.IsNaN(payload.AmountAOA) || math.IsInf(payload.AmountAOA, 0) {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Payload inválido"})
		return
	}

	var payment models.Payment
	if err := config.DB.First(&payment, "gateway_transaction_id = ?", payload.TransactionReference).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Transação não encontrada"})
		return
	}
	if payment.Method != "multicaixa_express" {
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "Método de pagamento incompatível"})
		return
	}
	if math.Abs(payment.AmountAOA-payload.AmountAOA) > 0.001 {
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "Montante da transação não corresponde"})
		return
	}
	if payment.Status == "paid" {
		if err := services.UpdateFleetPaymentStatus(payment.TripID, "paid"); err != nil {
			log.Printf("Pagamento já liquidado, mas a sincronização com fleet-api falhou: %v", err)
			c.JSON(http.StatusAccepted, gin.H{"status": "received", "paymentStatus": "paid", "fleetSync": "pending"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "received", "paymentStatus": "paid"})
		return
	}
	if payment.Status == "failed" && payload.Status != "PAID" {
		if err := services.UpdateFleetPaymentStatus(payment.TripID, "failed"); err != nil {
			log.Printf("Pagamento falhado, mas a sincronização com fleet-api falhou: %v", err)
			c.JSON(http.StatusAccepted, gin.H{"status": "received", "paymentStatus": "failed", "fleetSync": "pending"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "received", "paymentStatus": "failed"})
		return
	}
	if payment.Status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "Estado do pagamento não permite esta transição"})
		return
	}

	updates := map[string]interface{}{"status": "failed"}
	if payload.Status == "PAID" {
		updates["status"] = "paid"
		updates["paid_at"] = time.Now()
	}
	result := config.DB.Model(&models.Payment{}).
		Where("id = ? AND status = 'pending'", payment.ID).
		Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Não foi possível atualizar o pagamento"})
		return
	}
	if result.RowsAffected != 1 {
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "Pagamento alterado; verifique o estado atual"})
		return
	}

	status := "failed"
	if payload.Status == "PAID" {
		status = "paid"
	}
	if err := services.UpdateFleetPaymentStatus(payment.TripID, status); err != nil {
		log.Printf("Pagamento atualizado, mas a sincronização com fleet-api falhou: %v", err)
		c.JSON(http.StatusAccepted, gin.H{"status": "received", "paymentStatus": status, "fleetSync": "pending"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "received", "paymentStatus": status})
}

func validWebhookSignature(body []byte, signature, secret string) bool {
	signature = strings.TrimPrefix(signature, "sha256=")
	provided, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
}
