mod weathertweet {
    include!(concat!(env!("OUT_DIR"), "/weathertweet.rs"));
}

use actix_web::{web, App, HttpServer, HttpResponse, post};
use serde::{Deserialize, Serialize};
use std::sync::atomic::{AtomicU64, Ordering};
use reqwest::Client;
use std::sync::Arc;

#[derive(Debug, Serialize, Deserialize, Clone)]
struct WeatherTweet {
    municipality: String,
    temperature: i32,
    humidity: i32,
    weather: String,
}

#[derive(Debug, Serialize)]
struct ApiResponse {
    status: String,
    message: String,
}

// Contador de requests
static REQUEST_COUNT: AtomicU64 = AtomicU64::new(0);

#[post("/api/tweets")]
async fn receive_tweet(tweet: web::Json<WeatherTweet>) -> HttpResponse {
    let count = REQUEST_COUNT.fetch_add(1, Ordering::SeqCst);
    
    println!(
        "[Request #{}] Received tweet from {}: {}°C, {}% humidity, {} weather",
        count + 1,
        tweet.municipality,
        tweet.temperature,
        tweet.humidity,
        tweet.weather
    );

    // Validar municipios
    let valid_municipalities = vec!["mixco", "guatemala", "amatitlan", "chinautla"];
    if !valid_municipalities.contains(&tweet.municipality.as_str()) {
        return HttpResponse::BadRequest().json(ApiResponse {
            status: "error".to_string(),
            message: "Invalid municipality".to_string(),
        });
    }

    // Validar climas
    let valid_weathers = vec!["sunny", "cloudy", "rainy", "foggy"];
    if !valid_weathers.contains(&tweet.weather.as_str()) {
        return HttpResponse::BadRequest().json(ApiResponse {
            status: "error".to_string(),
            message: "Invalid weather".to_string(),
        });
    }

    // Enviar a Go service
    forward_to_go_service(&tweet).await;

    HttpResponse::Ok().json(ApiResponse {
        status: "success".to_string(),
        message: format!("Tweet processed (Request #{})", count + 1),
    })
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::init_from_env(env_logger::Env::new().default_filter_or("info"));

    println!("🚀 Starting Rust API server on 0.0.0.0:8080");

    HttpServer::new(|| {
        App::new()
            .service(receive_tweet)
    })
    .bind("0.0.0.0:8080")?
    .run()
    .await
}

// Función para enviar datos al servicio Go
async fn forward_to_go_service(tweet: &WeatherTweet) {
    let client = Client::new();
    let url = "http://go-orchestrator:8081/tweets";
    
    match client.post(url).json(tweet).send().await {
        Ok(response) => {
            if response.status().is_success() {
                println!("✅ Successfully forwarded to Go service");
            } else {
                println!("⚠️ Go service responded with: {}", response.status());
            }
        }
        Err(e) => {
            println!("❌ Error forwarding to Go service: {}", e);
        }
    }
}