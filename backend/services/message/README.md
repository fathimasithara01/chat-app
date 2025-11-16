# Chat Service (Go) - Microservice

Features:
- Fiber + WebSocket hub
- MongoDB storage for conversations & messages
- Redis for presence & caching
- Kafka for cross-service messaging/events
- JWT RS256 validation middleware
- Viper config (env + optional yaml)

Run:
1. Copy .env.example to .env and configure
2. go mod tidy
3. go run cmd/main.go

Notes:
- This service expects an external auth-service to issue JWT tokens.
- Improve: add Dockerfile, docker-compose, tests, metrics, and production observability.
