# Auth Service (Go + Fiber + Mongo + Redis + Twilio/Brevo)

1. Copy `.env.example` → `.env` and fill values (DO NOT commit `.env`).
2. Start dependencies (locally):
   - Redis (WSL): `redis-server --daemonize yes`
   - MongoDB (WSL): `sudo systemctl start mongod`
3. Run locally:
   cd backend/services/auth-service
   go mod tidy
   go run cmd/main.go

4. Test with Postman:
   POST http://localhost:8080/api/v1/auth/otp/request
   Body: { "phone": "+91xxxxxxxxxx" }

   POST http://localhost:8080/api/v1/auth/otp/verify
   Body: { "phone": "+91xxxxxxxxxx", "otp": "123456" }
