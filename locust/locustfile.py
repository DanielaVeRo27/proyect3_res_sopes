from locust import HttpUser, task, between
import json
import random

municipalities = ["mixco", "guatemala", "amatitlan", "chinautla"]
weathers = ["sunny", "cloudy", "rainy", "foggy"]

class WeatherTweetUser(HttpUser):
    wait_time = between(0.1, 0.5)

    @task
    def send_tweet(self):
        payload = {
            "municipality": random.choice(municipalities),
            "temperature": random.randint(15, 35),
            "humidity": random.randint(30, 90),
            "weather": random.choice(weathers)
        }
        
        self.client.post(
            "/api/tweets",
            json=payload,
            headers={"Content-Type": "application/json"}
        )
