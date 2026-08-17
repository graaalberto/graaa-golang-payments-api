package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

// ReceiptData contém todas as informações da corrida para emissão do recibo
type ReceiptData struct {
	ReceiptNumber   string    `json:"receiptNumber"`
	TripID          string    `json:"tripId"`
	PassengerName   string    `json:"passengerName"`
	PassengerPhone  string    `json:"passengerPhone"`
	DriverName      string    `json:"driverName"`
	VehiclePlate    string    `json:"vehiclePlate"`
	OriginAddress   string    `json:"originAddress"`
	DestAddress     string    `json:"destAddress"`
	DistanceKm      float64   `json:"distanceKm"`
	DurationMinutes int       `json:"durationMinutes"`
	AmountAOA       float64   `json:"amountAOA"`
	PaymentMethod   string    `json:"paymentMethod"`
	TransactionRef  string    `json:"transactionRef"`
	IssuedAt        time.Time `json:"issuedAt"`
}

// GenerateAndUploadReceipt cria o recibo e faz o upload para o MinIO (bucket: documents)
func GenerateAndUploadReceipt(data ReceiptData) (string, error) {
	// 1. Gera o conteúdo textual do recibo
	receiptContent := fmt.Sprintf(`=======================================================
               FROTAGO ANGOLA - RECIBO DE CORRIDA
=======================================================
Recibo Nº: %s
Data/Hora: %s

DADOS DA VIAGEM
-------------------------------------------------------
ID da Corrida: %s
Origem:        %s
Destino:       %s
Distância:     %.2f KM
Duração:       %d minutos

VIATURA & MOTORISTA
-------------------------------------------------------
Matrícula:     %s
Motorista:     %s

PAGAMENTO & TARIFA
-------------------------------------------------------
Passageiro:    %s (%s)
Método:        %s
Referência:    %s
TOTAL PAGO:    %.2f AOA (Kwanzas)
=======================================================
Obrigado por viajar com a FrotaGo Angola!
`,
		data.ReceiptNumber,
		data.IssuedAt.Format("02/01/2006 15:04:05"),
		data.TripID,
		data.OriginAddress,
		data.DestAddress,
		data.DistanceKm,
		data.DurationMinutes,
		data.VehiclePlate,
		data.DriverName,
		data.PassengerName,
		data.PassengerPhone,
		data.PaymentMethod,
		data.TransactionRef,
		data.AmountAOA,
	)

	// 2. Prepara o envio em multipart/form-data para a graaa-golang-minio-api-file
	minioURL := os.Getenv("MINIO_API_URL")
	if minioURL == "" {
		minioURL = "http://localhost:8081"
	}

	uploadURL := fmt.Sprintf("%s/api/files/upload", minioURL)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Define o bucket 'documents'
	_ = writer.WriteField("bucket", "documents")

	fileName := fmt.Sprintf("recibo_%s.txt", data.ReceiptNumber)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", fmt.Errorf("erro ao criar formulário de arquivo: %v", err)
	}

	_, err = part.Write([]byte(receiptContent))
	if err != nil {
		return "", fmt.Errorf("erro ao escrever dados do recibo: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("erro ao fechar writer: %v", err)
	}

	// 3. Executa a requisição HTTP POST para a API MinIO
	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		return "", fmt.Errorf("erro ao criar requisição de upload: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("falha ao enviar recibo para o MinIO: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("minio api retornou status code %d", resp.StatusCode)
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Sprintf("%s/api/files/documents/%s", minioURL, fileName), nil
	}

	return result.URL, nil
}
