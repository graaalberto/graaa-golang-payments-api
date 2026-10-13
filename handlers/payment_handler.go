package handlers

import (
	"fmt"
	"log"
	"math"
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
	if req.AmountAOA <= 0 || math.IsNaN(req.AmountAOA) || math.IsInf(req.AmountAOA, 0) {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Montante inválido"})
		return
	}
	if req.Method != "multicaixa_express" && req.Method != "cash" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Método de pagamento inválido"})
		return
	}
	var trip struct {
		FareAOA float64
	}
	if err := config.DB.Table("trips").Select("fare_aoa").Where("id = ?", req.TripID).Take(&trip).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Corrida não encontrada"})
		return
	}
	if math.Abs(req.AmountAOA-trip.FareAOA) > 0.001 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Montante diferente do valor da corrida"})
		return
	}
	req.AmountAOA = trip.FareAOA

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
		if req.CustomerPhone == "" {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Telefone do cliente obrigatório"})
			return
		}
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
		// A confirmação em dinheiro é feita por um operador autenticado.
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
	if c.GetString("userRole") != "admin" {
		userID := c.GetString("userID")
		var trip struct {
			DriverUserID     *string
			DispatcherUserID *string
		}
		if userID == "" || config.DB.Table("trips").
			Select("drivers.user_id AS driver_user_id, trips.dispatcher_user_id").
			Joins("LEFT JOIN drivers ON drivers.id = trips.driver_id").
			Where("trips.id = ?", payment.TripID).Take(&trip).Error != nil {
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acesso negado"})
			return
		}
		isDriver := trip.DriverUserID != nil && *trip.DriverUserID == userID
		isDispatcher := trip.DispatcherUserID != nil && *trip.DispatcherUserID == userID
		if !isDriver && !isDispatcher {
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acesso negado"})
			return
		}
	}
	c.JSON(http.StatusOK, payment)
}

func ConfirmCashPayment(c *gin.Context) {
	var payment models.Payment
	if err := config.DB.First(&payment, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Pagamento não encontrado"})
		return
	}
	if payment.Method != "cash" {
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "Apenas pagamentos em dinheiro podem ser confirmados por esta rota"})
		return
	}
	if payment.Status == "paid" {
		if err := services.UpdateFleetPaymentStatus(payment.TripID, "paid"); err != nil {
			log.Printf("Pagamento já liquidado, mas a sincronização com fleet-api falhou: %v", err)
			c.JSON(http.StatusAccepted, gin.H{"status": "success", "payment": payment, "fleetSync": "pending"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "payment": payment})
		return
	}
	if payment.Status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "Estado de pagamento não permite confirmação"})
		return
	}

	now := time.Now()
	result := config.DB.Model(&models.Payment{}).
		Where("id = ? AND method = ? AND status = ?", payment.ID, "cash", "pending").
		Updates(map[string]interface{}{"status": "paid", "paid_at": now})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Não foi possível confirmar o pagamento"})
		return
	}
	if result.RowsAffected != 1 {
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "Pagamento alterado; atualize e tente novamente"})
		return
	}
	payment.Status = "paid"
	payment.PaidAt = &now
	if err := services.UpdateFleetPaymentStatus(payment.TripID, "paid"); err != nil {
		log.Printf("Pagamento em dinheiro confirmado, mas a sincronização com fleet-api falhou: %v", err)
		c.JSON(http.StatusAccepted, gin.H{"status": "success", "payment": payment, "fleetSync": "pending"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "payment": payment})
}
