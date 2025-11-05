package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"notification-service/internal/consumer"
	"notification-service/internal/handler"
	"notification-service/internal/repository"
	"notification-service/internal/sender"
	"notification-service/internal/service"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

func main() {
	// 1. Setup logger
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	log.Info("Starting notification service...")

	// 2. Load .env file (check locally and one level up)
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

	if err := db.Ping(); err != nil {
		log.WithError(err).Fatal("Could not ping the database")
	}
	log.Info("Successfully connected to the PostgreSQL database")

	emailRepository := repository.NewPostgresEmailRepository(db)

	// 3. Create Email Sender
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	mailFrom := os.Getenv("MAIL_FROM")

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" || mailFrom == "" {
		log.Fatal("SMTP environment variables are not set")
	}

	emailSender := sender.NewSMTPEmailSender(smtpHost, smtpPort, smtpUser, smtpPass, mailFrom)

	// 4. Create Notification Service
	notificationService := service.NewNotificationService(emailSender, emailRepository)

	// 5. Create Handler
	purchaseHandler := handler.NewPurchaseHandler(notificationService)

	// 6. Setup Kafka Consumer
	kafkaServers := os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	if kafkaServers == "" {
		log.Fatal("KAFKA_BOOTSTRAP_SERVERS is not set")
	}
	kafkaServers = strings.Trim(kafkaServers, "\"")
	log.WithField("kafka_servers", kafkaServers).Info("Connecting to Kafka")

	configMap := &kafka.ConfigMap{
		"bootstrap.servers": kafkaServers,
		"group.id":          "notification_service_group",
		"auto.offset.reset": "earliest",
	}
	log.WithField("config", fmt.Sprintf("%+v", configMap)).Debug("Kafka consumer config")

	kafkaConsumer, err := kafka.NewConsumer(configMap)
	if err != nil {
		log.WithError(err).Fatal("Failed to create Kafka consumer")
	}

	topic := "successful_payments"
	kafkaConsumerWrapper, err := consumer.NewKafkaConsumer(kafkaConsumer, topic, purchaseHandler)
	if err != nil {
		log.WithError(err).Fatal("Failed to create Kafka consumer wrapper")
	}
	defer kafkaConsumerWrapper.Close()

	// 7. Graceful shutdown setup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// 8. Start consumer in goroutine
	go func() {
		if err := kafkaConsumerWrapper.Start(ctx); err != nil {
			log.WithError(err).Error("Kafka consumer stopped with error")
			cancel()
		}
	}()

	// 9. Wait for signal for graceful shutdown
	log.Info("Notification service started. Press Ctrl+C to stop.")
	<-sigchan
	log.Info("Shutting down notification service...")
	cancel()
	log.Info("Notification service stopped")
}
