package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"frotago-payments-api/config"
	"frotago-payments-api/handlers"
	"frotago-payments-api/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	if len(os.Getenv("JWT_SECRET")) < 32 {
		log.Fatal("JWT_SECRET deve ter pelo menos 32 caracteres")
	}

	// Conecta ao PostgreSQL (fleet_db)
	config.ConnectDB()

	mode := os.Getenv("GIN_MODE")
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	allowedOrigins := strings.FieldsFunc(os.Getenv("ALLOWED_ORIGINS"), func(r rune) bool {
		return r == ',' || r == ';'
	})
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:5173"}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "graaa-golang-payments-api",
			"database":  "fleet_db",
			"version":   "v1.0.0",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	api := r.Group("/api/payments")
	api.Use(middleware.JWTAuthMiddleware())
	{
		api.POST("/charge", middleware.RequireRole("admin", "member"), handlers.ChargeTrip)
		api.GET("/:id/status", handlers.GetPaymentStatus)
		api.GET("/wallet/:driverId", handlers.GetDriverWallet)
		api.POST("/:id/confirm-cash", middleware.RequireRole("admin"), handlers.ConfirmCashPayment)
	}
	r.POST("/api/payments/webhook/multicaixa", handlers.MulticaixaWebhook)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	log.Printf("💳 FrotaGo Payments API (Multicaixa Express) rodando na porta :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Erro ao iniciar servidor de pagamentos: %v", err)
	}
}
