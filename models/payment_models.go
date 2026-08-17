package models

import "time"

// Payment representa a transação financeira da corrida
type Payment struct {
	ID                   string     `gorm:"primaryKey;type:varchar(64)" json:"id"`
	TripID               string     `gorm:"not null;type:varchar(64)" json:"tripId"`
	AmountAOA            float64    `gorm:"not null" json:"amountAOA"`
	Method               string     `gorm:"not null;type:varchar(40)" json:"method"`          // multicaixa_express, cash, card_pos
	Status               string     `gorm:"default:'pending';type:varchar(30)" json:"status"` // pending, paid, failed, refunded
	GatewayTransactionID string     `gorm:"type:varchar(100)" json:"gatewayTransactionId"`
	CustomerPhone        string     `gorm:"type:varchar(30)" json:"customerPhone"`
	ReceiptPDFURL        string     `json:"receiptPdfUrl"` // URL MinIO (bucket: documents)
	PaidAt               *time.Time `json:"paidAt"`
	CreatedAt            time.Time  `json:"createdAt"`
}

// DriverWallet representa o saldo e comissão do motorista
type DriverWallet struct {
	DriverID         string    `gorm:"primaryKey;type:varchar(64)" json:"driverId"`
	AvailableBalance float64   `gorm:"default:0.0" json:"availableBalanceAOA"`
	PendingBalance   float64   `gorm:"default:0.0" json:"pendingBalanceAOA"`
	TotalEarnedAOA   float64   `gorm:"default:0.0" json:"totalEarnedAOA"`
	CompanyFeeAOA    float64   `gorm:"default:0.0" json:"companyFeeAOA"`
	UpdatedAt        time.Time `json:"updatedAt"`
}
