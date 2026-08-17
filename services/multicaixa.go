package services

import (
	"fmt"
	"math/rand"
	"time"
)

type MulticaixaChargeRequest struct {
	PhoneNumber string  `json:"phoneNumber"`
	AmountAOA   float64 `json:"amountAOA"`
	TripID      string  `json:"tripId"`
}

type MulticaixaChargeResponse struct {
	TransactionReference string    `json:"transactionReference"`
	Status               string    `json:"status"` // PENDING_USER_AUTHORIZATION, SUCCESS, FAILED
	ExpiresAt            time.Time `json:"expiresAt"`
	Message              string    `json:"message"`
}

// ProcessMulticaixaExpress envia o push de autorização para o telemóvel do passageiro
func ProcessMulticaixaExpress(req MulticaixaChargeRequest) (*MulticaixaChargeResponse, error) {
	// Simulação do Gateway EMIS GPO Angola
	ref := fmt.Sprintf("MCX_%d_%d", time.Now().Unix(), rand.Intn(9999))
	return &MulticaixaChargeResponse{
		TransactionReference: ref,
		Status:               "PENDING_USER_AUTHORIZATION",
		ExpiresAt:            time.Now().Add(5 * time.Minute),
		Message:              fmt.Sprintf("Pedido de pagamento de %.2f Kz enviado para o telemóvel %s via Multicaixa Express", req.AmountAOA, req.PhoneNumber),
	}, nil
}
