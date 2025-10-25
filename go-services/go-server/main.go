package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WeatherTweet struct {
	Municipality string `json:"municipality"`
	Temperature  int32  `json:"temperature"`
	Humidity     int32  `json:"humidity"`
	Weather      string `json:"weather"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

var requestCount int64

// Clientes gRPC (usar nombres diferentes para evitar conflictos)
var kafkaClient KafkaWriterClient
var rabbitmqClient RabbitmqWriterClient

func init() {
	// Conectar a Kafka Writer por gRPC
	kafkaConn, err := grpc.Dial("kafka-writer:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Warning: Could not connect to Kafka Writer: %v\n", err)
	} else {
		kafkaClient = NewKafkaWriterClient(kafkaConn)
		fmt.Println("✅ Connected to Kafka Writer (gRPC)")
	}

	// Conectar a RabbitMQ Writer por gRPC
	rabbitmqConn, err := grpc.Dial("rabbitmq-writer:50053", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Warning: Could not connect to RabbitMQ Writer: %v\n", err)
	} else {
		rabbitmqClient = NewRabbitmqWriterClient(rabbitmqConn)
		fmt.Println("✅ Connected to RabbitMQ Writer (gRPC)")
	}
}

func handleTweets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var tweet WeatherTweet
	err := json.NewDecoder(r.Body).Decode(&tweet)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	count := atomic.AddInt64(&requestCount, 1)
	fmt.Printf("[Go Orchestrator - Request #%d] Received from Rust: %s, %d°C, %d%% humidity, %s weather\n",
		count, tweet.Municipality, tweet.Temperature, tweet.Humidity, tweet.Weather)

	// Publicar a Kafka via gRPC
	go publishToKafkaViaGRPC(&tweet)

	// Publicar a RabbitMQ via gRPC
	go publishToRabbitMQViaGRPC(&tweet)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Status:  "success",
		Message: fmt.Sprintf("Processed and published (Request #%d)", count),
	})
}

func publishToKafkaViaGRPC(tweet *WeatherTweet) {
	if kafkaClient == nil {
		fmt.Println("  ⚠️ Kafka Writer not connected")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &KafkaMessage{
		Municipality: tweet.Municipality,
		Temperature:  tweet.Temperature,
		Humidity:     tweet.Humidity,
		Weather:      tweet.Weather,
	}

	_, err := kafkaClient.PublishToKafka(ctx, req)
	if err != nil {
		fmt.Printf("  ❌ Error calling Kafka Writer: %v\n", err)
		return
	}
	fmt.Println("  ✅ Published to Kafka (via gRPC)")
}

func publishToRabbitMQViaGRPC(tweet *WeatherTweet) {
	if rabbitmqClient == nil {
		fmt.Println("  ⚠️ RabbitMQ Writer not connected")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &RabbitmqMessage{
		Municipality: tweet.Municipality,
		Temperature:  tweet.Temperature,
		Humidity:     tweet.Humidity,
		Weather:      tweet.Weather,
	}

	_, err := rabbitmqClient.PublishToRabbitMQ(ctx, req)
	if err != nil {
		fmt.Printf("  ❌ Error calling RabbitMQ Writer: %v\n", err)
		return
	}
	fmt.Println("  ✅ Published to RabbitMQ (via gRPC)")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func main() {
	fmt.Println("🚀 Starting Go Orchestrator Service on 0.0.0.0:8081")

	http.HandleFunc("/tweets", handleTweets)
	http.HandleFunc("/health", handleHealth)

	go startGRPCServer()

	err := http.ListenAndServe("0.0.0.0:8081", nil)
	if err != nil {
		log.Fatalf("HTTP Server error: %v", err)
	}
}

func startGRPCServer() {
	listener, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Failed to listen on port 50051: %v", err)
	}

	grpcServer := grpc.NewServer()
	fmt.Println("📡 gRPC Server listening on 0.0.0.0:50051")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("gRPC Server error: %v", err)
	}
}
