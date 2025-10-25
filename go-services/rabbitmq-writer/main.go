package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync/atomic"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

type WeatherTweet struct {
	Municipality string `json:"municipality"`
	Temperature  int32  `json:"temperature"`
	Humidity     int32  `json:"humidity"`
	Weather      string `json:"weather"`
}

var messageCount int64
var channel *amqp.Channel

// Implementación del servidor gRPC
type rabbitmqWriterServer struct {
	UnimplementedRabbitmqWriterServer
}

func (s *rabbitmqWriterServer) PublishToRabbitMQ(ctx context.Context, req *RabbitmqMessage) (*RabbitmqResponse, error) {
	tweet := WeatherTweet{
		Municipality: req.Municipality,
		Temperature:  req.Temperature,
		Humidity:     req.Humidity,
		Weather:      req.Weather,
	}

	count := atomic.AddInt64(&messageCount, 1)
	data, err := json.Marshal(tweet)
	if err != nil {
		return &RabbitmqResponse{
			Status:  "error",
			Message: fmt.Sprintf("Error marshaling: %v", err),
		}, err
	}

	err = channel.Publish(
		"",               // exchange
		"weather-tweets", // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         data,
		},
	)

	if err != nil {
		fmt.Printf("❌ Error publishing to RabbitMQ: %v\n", err)
		return &RabbitmqResponse{
			Status:  "error",
			Message: fmt.Sprintf("Error publishing: %v", err),
		}, err
	}

	fmt.Printf("[RabbitMQ Writer - Message #%d] Published: %s\n", count, string(data))

	return &RabbitmqResponse{
		Status:  "success",
		Message: "Published to RabbitMQ",
	}, nil
}

func main() {
	fmt.Println("🚀 Starting Go RabbitMQ Writer Service")

	// Conectar a RabbitMQ
	conn, err := amqp.Dial("amqp://default_user_pSqIHyPHkVx5_JyTINs:OKVZmdiOw6vAteDhXJ7z3sTIwFtguUCs@rabbitmq-cluster:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	channel, err = conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer channel.Close()

	// Declarar la cola como durable
	_, err = channel.QueueDeclare(
		"weather-tweets",
		true,  // durable - cambiar a true
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	fmt.Println("✅ Connected to RabbitMQ on rabbitmq-cluster:5672")

	// Iniciar servidor gRPC
	listener, err := net.Listen("tcp", "0.0.0.0:50053")
	if err != nil {
		log.Fatalf("Failed to listen on port 50053: %v", err)
	}

	grpcServer := grpc.NewServer()
	RegisterRabbitmqWriterServer(grpcServer, &rabbitmqWriterServer{})

	fmt.Println("📡 gRPC Server listening on 0.0.0.0:50053")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("gRPC Server error: %v", err)
	}
}
