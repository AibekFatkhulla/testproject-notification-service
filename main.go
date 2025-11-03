package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/smtp"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"notification-service/internal/repository"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jordan-wright/email"
	_ "github.com/lib/pq"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

// Определяем структуру, в которую будем распаковывать JSON из Kafka
type PurchaseInfo struct {
	UserID         string `json:"user_id"`
	UserEmail      string `json:"user_email"`
	CoinsPurchased int    `json:"coins_purchased"`
	TransactionID  string `json:"transaction_id"`
	// ... остальные поля можно опустить, если они не нужны для email
}

func main() {
	// 1. Настройка логгера
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	log.Info("Starting notification service...")

	// 2. Загрузка .env (ищем и локально, и на уровень выше)
	if err := godotenv.Load("../.env"); err != nil {
		log.Warn("Could not load .env file.")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	// Use a separate migrations table to avoid conflicts with the core service migrations
	migrationDBURL := dbURL
	if strings.Contains(dbURL, "?") {
		migrationDBURL = dbURL + "&x-migrations-table=notification_schema_migrations"
	} else {
		migrationDBURL = dbURL + "?x-migrations-table=notification_schema_migrations"
	}

	m, err := migrate.New("file://db/migrations", migrationDBURL)
	if err != nil {
		log.WithError(err).Fatal("Could not create migration instance")
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.WithError(err).Fatal("Could not apply migration")
	}
	log.Info("Database migration successfully applied")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.WithError(err).Fatal("Could not connect to database")
	}
	defer db.Close()

	emailRepository := repository.NewPostgresEmailRepository(db)

	kafkaServers := os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	if kafkaServers == "" {
		log.Fatal("KAFKA_BOOTSTRAP_SERVERS is not set")
	}
	// Убираем кавычки, если они есть
	kafkaServers = strings.Trim(kafkaServers, "\"")
	log.WithField("kafka_servers", kafkaServers).Info("Connecting to Kafka")

	// 3. Создание и настройка "Потребителя" (Consumer)
	configMap := &kafka.ConfigMap{
		"bootstrap.servers": kafkaServers,
		"group.id":          "notification_service_group",
		"auto.offset.reset": "earliest",
	}
	log.WithField("config", fmt.Sprintf("%+v", configMap)).Debug("Kafka consumer config")
	consumer, err := kafka.NewConsumer(configMap)
	if err != nil {
		log.WithError(err).Fatal("Failed to create Kafka consumer")
	}
	defer consumer.Close()

	// 4. Подписываемся на нужный нам топик
	topic := "successful_payments"
	if err := consumer.SubscribeTopics([]string{topic}, nil); err != nil {
		log.WithError(err).Fatal("Failed to subscribe to topic")
	}
	log.WithField("topic", topic).Info("Subscribed to Kafka topic")

	// 5. Запускаем бесконечный цикл для чтения сообщений
	// (Он будет работать, пока мы не остановим приложение через Ctrl+C)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	run := true
	for run {
		select {
		case sig := <-sigchan:
			log.Infof("Caught signal %v: terminating", sig)
			run = false
		default:
			// Ждем новое сообщение до 100 миллисекунд
			ev := consumer.Poll(100)
			if ev == nil {
				continue // Сообщений нет, начинаем цикл заново
			}

			switch e := ev.(type) {
			case *kafka.Message:
				var purchase PurchaseInfo
				if err := json.Unmarshal(e.Value, &purchase); err != nil {
					log.WithError(err).Error("Failed to unmarshal purchase info")
					continue
				}

				logCtx := log.WithFields(log.Fields{
					"transaction_id": purchase.TransactionID,
					"user_id":        purchase.UserID,
				})
				logCtx.Info("Processing purchase event for email notification")

				sendAndLogEmail(emailRepository, purchase)

			case kafka.Error:
				log.WithError(e).Error("Kafka error")
				if e.IsFatal() {
					run = false
				}
			}
		}
	}
}

func sendAndLogEmail(repo repository.EmailRepository, purchase PurchaseInfo) {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("MAIL_FROM")

	if host == "" || port == "" || user == "" || pass == "" || from == "" {
		log.Error("SMTP environment variables are not set. Email not sent.")
		return
	}

	auth := smtp.PlainAuth("", user, pass, host)
	addr := fmt.Sprintf("%s:%s", host, port)

	e := email.NewEmail()
	e.From = from
	e.To = []string{purchase.UserEmail}
	e.Subject = "Покупка монет успешно завершена!"
	e.Text = []byte(fmt.Sprintf(
		"Здравствуйте!\n\nВы успешно приобрели %d монет.\nID вашей транзакции: %s\n\nСпасибо за покупку!",
		purchase.CoinsPurchased,
		purchase.TransactionID,
	))

	err := e.Send(addr, auth)

	logEntry := repository.EmailLog{
		TransactionID:  purchase.TransactionID,
		RecipientEmail: purchase.UserEmail,
		Subject:        e.Subject,
	}

	if err != nil {
		log.WithError(err).Error("Failed to send confirmation email via SMTP")
		logEntry.Status = repository.StatusFailed
		logEntry.ErrorMessage = sql.NullString{String: err.Error(), Valid: true}
	} else {
		log.WithField("email", purchase.UserEmail).Info("Confirmation email sent successfully via SMTP")
		logEntry.Status = repository.StatusSent
	}

	if err := repo.SaveLog(logEntry); err != nil {
		log.WithError(err).Error("Failed to save email log to database")
	}
}
