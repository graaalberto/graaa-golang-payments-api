💳 API 4: Gateway de Pagamentos & Cobranças em Golang (graaa-golang-payments-api)

## Configuração de segurança

- Defina `JWT_SECRET` igual ao segredo do auth API e com pelo menos 32 bytes. Cobranças, consulta de estado e carteiras exigem JWT de acesso; um motorista só pode consultar a própria carteira e os pagamentos das corridas associadas.
- Defina `ALLOWED_ORIGINS` como a lista separada por vírgulas das origens exatas do frontend; sem configuração, apenas `http://localhost:5173` é autorizado.
- Defina `MULTICAIXA_WEBHOOK_SECRET` como segredo aleatório de pelo menos 32 bytes, partilhado somente com o provedor. O callback exige `X-Multicaixa-Signature: sha256=<HMAC-SHA256 do corpo HTTP bruto>`, valida o montante e só permite transições a partir de `pending`.
- Defina `FLEET_PAYMENT_TOKEN` com um segredo aleatório independente e partilhe-o apenas com a fleet API. Se a sincronização estiver indisponível, a cobrança permanece confirmada e a resposta informa `fleetSync: pending`; callbacks repetidos tentam sincronizar novamente.
- Pagamentos em dinheiro permanecem `pending` até um administrador autenticado chamar `POST /api/payments/:id/confirm-cash`. Não marque pagamentos como liquidados a partir do pedido de cobrança do cliente.
- A implementação Multicaixa neste repositório é uma simulação. Uma assinatura HMAC só autentica um provedor que detenha o segredo; antes de produção, integrar com a confirmação oficial EMIS e validar a transação diretamente com o gateway.

Focada no mercado financeiro angolano e internacional:

    Multicaixa Express (EMIS GPO / QR Code / Referência Bancária): Débito direto em Kwanzas (AOA) via número de telefone do passageiro.

    Carteira Digital do Motorista (Driver Digital Wallet & Cash Reconciliation): Fecho de caixa diário, comissões da central da frota (ex.: 15%) e saldo retido.

    Webhooks de Confirmação de Pagamento: Notifica a API Central da Frota e conclui a corrida assim que o cliente aprovar o pagamento no telemóvel.

    Recibos em PDF Automáticos no MinIO: Armazenamento do comprovativo de pagamento no bucket documents da API MinIO.

📁 Estrutura da Pasta: backend-payments-golang
code Text

backend-payments-golang/
├── .env
├── go.mod
├── main.go
├── config/
│   └── database.go          # Conexão com o fleet_db (tabela payments)
├── models/
│   └── payment_models.go    # Structs de Pagamento, Carteira e Transações
├── services/
│   ├── multicaixa.go        # Simulação e Integração com Multicaixa Express / EMIS
│   └── receipt_generator.go # Geração de recibo de corrida
└── handlers/
    ├── payment_handler.go   # Iniciação de cobrança e consulta de transação
    ├── webhook_handler.go   # Webhook para confirmação de pagamento
    └── wallet_handler.go    # Fecho de caixa e repasse para motoristas

```
🚀 Como Executar no PowerShell (Windows 11):

    Crie a pasta backend-payments-golang e adicione os arquivos acima.

    No PowerShell:

code Powershell

cd backend-payments-golang
go mod tidy
go run main.go

Quando você rodar, verá a mensagem:
💳 FrotaGo Payments API (Multicaixa Express) rodando na porta :8083


📊 Painel Atual dos Seus Microserviços em Execução:
#	Microserviço / API	Porta	Status	Responsabilidade Principal
1	graaa-golang-auth-api	:8080	🟢 Ativo	Autenticação JWT, Usuários e RBAC
2	graaa-golang-minio-api-file	:8081	🟢 Ativo	Storage S3/MinIO (Avatares, Recibos, Avarias)
3	graaa-golang-fleet-api	:8082	🟢 Ativo	Gestão de Frota, Corridas & WebSockets GPS
4	graaa-golang-payments-api	:8083	🟢 Ativo	Pagamentos Multicaixa Express, Carteira & Recibos
