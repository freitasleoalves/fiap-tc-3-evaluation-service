package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

// Contexto global para o Redis
var ctx = context.Background()

// App struct para injeção de dependência
type App struct {
	RedisClient         *redis.Client
	MsgSender           MessageSender
	HttpClient          *http.Client
	FlagServiceURL      string
	TargetingServiceURL string
}

func main() {
	_ = godotenv.Load() // Carrega .env para dev local

	// --- Configuração ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8004"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL deve ser definida (ex: redis://localhost:6379)")
	}

	flagSvcURL := os.Getenv("FLAG_SERVICE_URL")
	if flagSvcURL == "" {
		log.Fatal("FLAG_SERVICE_URL deve ser definida")
	}

	targetingSvcURL := os.Getenv("TARGETING_SERVICE_URL")
	if targetingSvcURL == "" {
		log.Fatal("TARGETING_SERVICE_URL deve ser definida")
	}

	// --- Inicializa Clientes ---
	
	// Cliente Redis
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Não foi possível parsear a URL do Redis: %v", err)
	}
	rdb := redis.NewClient(opt)
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("Não foi possível conectar ao Redis: %v", err)
	}
	log.Println("Conectado ao Redis com sucesso!")

	// Inicializa o MessageSender baseado no CLOUD_PROVIDER
	var msgSender MessageSender
	cloudProvider := os.Getenv("CLOUD_PROVIDER") // "azure" ou "aws" (default)

	if cloudProvider == "azure" {
		// Azure Service Bus
		sbConnStr := os.Getenv("AZURE_SERVICEBUS_CONNECTION_STRING")
		sbQueueName := os.Getenv("AZURE_SERVICEBUS_QUEUE_NAME")
		if sbConnStr == "" || sbQueueName == "" {
			log.Fatal("AZURE_SERVICEBUS_CONNECTION_STRING e AZURE_SERVICEBUS_QUEUE_NAME devem ser definidos para azure")
		}
		sender, err := NewServiceBusSender(sbConnStr, sbQueueName)
		if err != nil {
			log.Fatalf("Não foi possível criar Service Bus sender: %v", err)
		}
		msgSender = sender
		log.Println("Cliente Azure Service Bus inicializado com sucesso.")
	} else {
		// AWS SQS (default - para dev local com LocalStack)
		sqsQueueURL := os.Getenv("AWS_SQS_URL")
		awsRegion := os.Getenv("AWS_REGION")
		if sqsQueueURL == "" {
			log.Println("Atenção: AWS_SQS_URL não definida. Eventos não serão enviados.")
		} else {
			if awsRegion == "" {
				log.Fatal("AWS_REGION deve ser definida para usar SQS")
			}
			awsCfg := &aws.Config{Region: aws.String(awsRegion)}
			if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
				awsCfg.Endpoint = aws.String(endpoint)
				awsCfg.S3ForcePathStyle = aws.Bool(true)
			}
			sess, err := session.NewSession(awsCfg)
			if err != nil {
				log.Fatalf("Não foi possível criar sessão AWS: %v", err)
			}
			msgSender = &SQSSender{SqsSvc: sqs.New(sess), QueueURL: sqsQueueURL}
			log.Println("Cliente SQS inicializado com sucesso.")
		}
	}

	// Cliente HTTP (com timeout)
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Cria a instância da App
	app := &App{
		RedisClient:         rdb,
		MsgSender:           msgSender,
		HttpClient:          httpClient,
		FlagServiceURL:      flagSvcURL,
		TargetingServiceURL: targetingSvcURL,
	}

	// --- Rotas ---
	mux := http.NewServeMux()
	mux.HandleFunc("/health", app.healthHandler)
	mux.HandleFunc("/evaluate", app.evaluationHandler)

	log.Printf("Serviço de Avaliação (Go) rodando na porta %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}