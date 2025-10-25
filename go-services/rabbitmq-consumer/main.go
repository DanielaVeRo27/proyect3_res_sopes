package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type WeatherTweet struct {
	Municipality string `json:"municipality"`
	Temperature  int32  `json:"temperature"`
	Humidity     int32  `json:"humidity"`
	Weather      string `json:"weather"`
}

func main() {
	fmt.Println("🚀 Starting RabbitMQ Consumer Service")

	// Conectar a RabbitMQ
	conn, err := amqp.Dial("amqp://default_user_pSqIHyPHkVx5_JyTINs:OKVZmdiOw6vAteDhXJ7z3sTIwFtguUCs@rabbitmq-cluster:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declarar la cola
	q, err := ch.QueueDeclare(
		"weather-tweets",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	fmt.Println("✅ Connected to RabbitMQ")

	// Conectar a Valkey
	rdb := redis.NewClient(&redis.Options{
		Addr: "valkey-redis-master:6379",
	})
	defer rdb.Close()

	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Valkey: %v", err)
	}
	fmt.Println("✅ Connected to Valkey")

	// Consumir mensajes
	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	for d := range msgs {
		fmt.Println("📥 Message received from queue")
		fmt.Println("📥 Message received from queue")
		fmt.Println("📥 Message received from queue")
		fmt.Println("📥 Message received from queue")
		var tweet WeatherTweet
		err := json.Unmarshal(d.Body, &tweet)
		fmt.Printf("✅ Unmarshaled successfully: %+v\n", tweet)
		if err != nil {
			fmt.Printf("Error unmarshaling message: %v\n", err)
			continue
		}

		fmt.Printf("[RabbitMQ Consumer] Received: %s - %s weather\n", tweet.Municipality, tweet.Weather)

		// Guardar en Valkey
		key := fmt.Sprintf("weather:%s:%s", tweet.Municipality, tweet.Weather)
		rdb.Incr(context.Background(), key)

		temperatura_Key := fmt.Sprintf("temperatura:%s:%s", tweet.Municipality, tweet.Temperature)
		rdb.Incr(context.Background(), temperatura_Key)

		municipalityKey := fmt.Sprintf("municipality:%s:count", tweet.Municipality)
		rdb.Incr(context.Background(), municipalityKey)

		// Guardar temperatura y humedad por municipio (no por valor)
		temperatureKey := fmt.Sprintf("municipality:%s:temperature_sum", tweet.Municipality)
		rdb.IncrBy(context.Background(), temperatureKey, int64(tweet.Temperature))

		humidityKey := fmt.Sprintf("municipality:%s:humidity_sum", tweet.Municipality)
		rdb.IncrBy(context.Background(), humidityKey, int64(tweet.Humidity))

		temperatura_Prom_Key := fmt.Sprintf("municipality:%s:temperature_avg/count", tweet.Municipality)
		rdb.IncrBy(context.Background(), temperatura_Prom_Key, int64(tweet.Temperature))

		temperatureKey2 := fmt.Sprintf("municipality:%s:temperature", tweet.Municipality)
		rdb.Set(context.Background(), temperatureKey2, tweet.Temperature, 0)

		humidityKey2 := fmt.Sprintf("municipality:%s:humidity", tweet.Municipality)
		rdb.Set(context.Background(), humidityKey2, tweet.Humidity, 0)

		// ------ Variables o KEY nuevas para las graficas en el dashboard
		climaKey := fmt.Sprintf("clima:%s:count", tweet.Weather)
		rdb.Incr(context.Background(), climaKey)

		// 🌤️ Condición climática (contador por tipo)
		conditionKey := fmt.Sprintf("weather:%s:condition:%s", tweet.Municipality, tweet.Weather)
		rdb.Incr(context.Background(), conditionKey)

		// 🌡️ Suma total de temperaturas
		temperatureSumKey := fmt.Sprintf("municipality:%s:temperature:avg", tweet.Municipality)
		rdb.IncrBy(context.Background(), temperatureSumKey, int64(tweet.Temperature))

		// -----> Calculamos el promedio de temperatura promedio por municipio
		sumKey := fmt.Sprintf("municipality:%s:temperature:sum", tweet.Municipality)
		countKey := fmt.Sprintf("municipality:%s:temperature:count", tweet.Municipality)
		avgKey := fmt.Sprintf("municipality:%s:temperature:avg", tweet.Municipality)

		// Actualiza sum y count de forma atómica
		pipe := rdb.TxPipeline()
		pipe.IncrBy(context.Background(), sumKey, int64(tweet.Temperature))
		pipe.Incr(context.Background(), countKey)
		vals, _ := pipe.Exec(context.Background())
		println(vals)

		// Luego lee sum y count (o usa MGET) y calcula avg
		sum, _ := rdb.Get(context.Background(), sumKey).Int64()
		count, _ := rdb.Get(context.Background(), countKey).Int64()
		if count > 0 {
			avg := float64(sum) / float64(count)
			// Guarda el promedio como string (Grafana lo lee con GET)
			rdb.Set(context.Background(), avgKey, fmt.Sprintf("%.2f", avg), 0)
		}

		// --------------------> humedad promedio <--------------------
		// -----> Calculamos el promedio de temperatura promedio por municipio
		sumKey2 := fmt.Sprintf("municipality:%s:humedad:sum", tweet.Municipality)
		countKey2 := fmt.Sprintf("municipality:%s:humedad:count", tweet.Municipality)
		avgKey2 := fmt.Sprintf("municipality:%s:humedad:avg", tweet.Municipality)

		// Actualiza sum y count de forma atómica
		pipe2 := rdb.TxPipeline()
		pipe2.IncrBy(context.Background(), sumKey2, int64(tweet.Humidity))
		pipe2.Incr(context.Background(), countKey2)
		vals2, _ := pipe2.Exec(context.Background())
		println(vals2)

		// Luego lee sum y count (o usa MGET) y calcula avg
		sum2, _ := rdb.Get(context.Background(), sumKey2).Int64()
		count2, _ := rdb.Get(context.Background(), countKey2).Int64()
		if count2 > 0 {
			avg := float64(sum2) / float64(count2)
			// Guarda el promedio como string (Grafana lo lee con GET)
			rdb.Set(context.Background(), avgKey2, fmt.Sprintf("%.2f", avg), 0)
		}

		// --------------------- Max y Min -----------------------
		ctx := context.Background()

		tMaxKey := "metrics:temperature:max"
		tMinKey := "metrics:temperature:min"

		// 1) toma la temperatura del mensaje y casteala a int64
		temp := int64(tweet.Temperature)

		// ---------- MAX ----------
		tMax, err := rdb.Get(ctx, tMaxKey).Int64()
		if err != nil && err != redis.Nil {
			log.Println("get tMax:", err)
		}
		// si la clave no existe (redis.Nil) o temp es mayor, actualiza
		if err == redis.Nil || temp > tMax {
			if err := rdb.Set(ctx, tMaxKey, temp, 0).Err(); err != nil {
				log.Println("set tMax:", err)
			}
		}

		// ---------- MIN ----------
		tMinStr, err := rdb.Get(ctx, tMinKey).Result()
		if err != nil && err != redis.Nil {
			log.Println("get tMin:", err)
		}
		if err == redis.Nil || tMinStr == "" {
			// inicializa con el primer valor
			_ = rdb.Set(ctx, tMinKey, temp, 0).Err()
		} else {
			tMin, _ := strconv.ParseInt(tMinStr, 10, 64)
			if temp < tMin {
				_ = rdb.Set(ctx, tMinKey, temp, 0).Err()
			}
		}

		// --------------------- Max y Min HUMEDAD -----------------------
		hMaxKey := "metrics:humedad:max"
		hMinKey := "metrics:humedad:min"

		// 1) toma la humedad del mensaje y casteala a int64
		hum := int64(tweet.Humidity)
		// ---------- MAX ----------
		hMax, err := rdb.Get(ctx, hMaxKey).Int64()
		if err != nil && err != redis.Nil {
			log.Println("get hMax:", err)
		}
		// si la clave no existe (redis.Nil) o hum es mayor, actualiza
		if err == redis.Nil || hum > hMax {
			if err := rdb.Set(ctx, hMaxKey, hum, 0).Err(); err != nil {
				log.Println("set hMax:", err)
			}
		}

		// ---------- MIN ----------
		hMinStr, err := rdb.Get(ctx, hMinKey).Result()
		if err != nil && err != redis.Nil {
			log.Println("get hMin:", err)
		}
		if err == redis.Nil || hMinStr == "" {
			// inicializa con el primer valor
			_ = rdb.Set(ctx, hMinKey, hum, 0).Err()
		} else {
			hMin, _ := strconv.ParseInt(hMinStr, 10, 64)
			if hum < hMin {
				_ = rdb.Set(ctx, hMinKey, hum, 0).Err()
			}
		}

		// -------------> parte de las graficas del clima
		muni := tweet.Municipality
		conditions := []string{"sunny", "rainy", "cloudy", "foggy"}

		var (
			mostCommonWeather  string
			leastCommonWeather string
			highestCount       int64 = -1
			lowestCount        int64 = 999999999
		)

		for _, c := range conditions {
			key := fmt.Sprintf("weather:%s:condition:%s", muni, c)
			count, err := rdb.Get(ctx, key).Int64()
			if err == redis.Nil {
				count = 0
			} else if err != nil {
				log.Println("Error leyendo", key, ":", err)
				continue
			}

			if count > highestCount {
				highestCount = count
				mostCommonWeather = c
			}
			if count < lowestCount {
				lowestCount = count
				leastCommonWeather = c
			}
		}

		// Guardar resultados finales
		rdb.Set(ctx, fmt.Sprintf("municipality:%s:weather:most_common", muni), mostCommonWeather, 0)
		rdb.Set(ctx, fmt.Sprintf("municipality:%s:weather:least_common", muni), leastCommonWeather, 0)

		fmt.Println(" ------------ Data saved to Valkey: ------------------")
		fmt.Println("Mas comun Weather:", mostCommonWeather)
		fmt.Println("menos comun Weather:", leastCommonWeather)
		fmt.Println(" -------------------------------------- ")

		fmt.Println(" ------------ Data saved to Valkey: ------------------")
		fmt.Println("Weather Key ", key)
		fmt.Println("Municipality Key ", municipalityKey)
		fmt.Println("Temperature Key ", temperatureKey)
		fmt.Println("Humidity Key ", humidityKey)
		fmt.Println("Condition Key ", conditionKey)
		fmt.Println("Temperature Sum Key ", temperatureSumKey)
		fmt.Println()
		fmt.Println(" -------------------------------------- ")

	}
}
