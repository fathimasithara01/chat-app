WebSocket service (Option A - Redis Pub/Sub)

Run:
  docker-compose up --build

Connect:
  ws://localhost:8081/ws?token=<jwt>

Events (client -> server):
  { "type":"message", "body": {...}, "id":"msgid", "to":"user-id" }
  { "type":"typing", "to":"user-id" }
  { "type":"read", "to":"user-id", "body": { "message_id": "..." } }

Notes:
- Messages are published to Redis `msg:incoming` channel; message-service should subscribe and persist.
- Presence uses keys `presence:<userID>`.
- Server expects JWT with 'sub' claim containing user id.
