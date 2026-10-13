package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func UpdateFleetPaymentStatus(tripID, status string) error {
	token := os.Getenv("FLEET_PAYMENT_TOKEN")
	if len(token) < 32 {
		return fmt.Errorf("FLEET_PAYMENT_TOKEN não está configurado")
	}
	baseURL := os.Getenv("FLEET_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8082"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !(parsed.Scheme == "http" && (parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1"))) {
		return fmt.Errorf("FLEET_API_URL deve usar HTTPS, exceto em localhost")
	}

	body, err := json.Marshal(map[string]string{"status": status})
	if err != nil {
		return fmt.Errorf("falha ao serializar estado de pagamento: %w", err)
	}
	endpoint := fmt.Sprintf("%s/api/internal/trips/%s/payment-status", strings.TrimRight(baseURL, "/"), url.PathEscape(tripID))
	req, err := http.NewRequest(http.MethodPatch, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("falha ao criar pedido à fleet API: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Token", token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("falha ao atualizar estado de pagamento na fleet API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("fleet API respondeu HTTP %d ao atualizar estado de pagamento", resp.StatusCode)
	}
	return nil
}
