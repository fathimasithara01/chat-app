# Chat-App — Real-Time Messaging Platform (Go + Microservices)

A production-grade real-time messaging platform built using **Go (Fiber), WebSockets, Redis, Kafka, and MongoDB**, structured as fully isolated microservices following clean architectural principles.

Designed for **high scalability, low latency, and distributed fault tolerance**, this system models how modern real-time applications such as Slack, Discord, and WhatsApp handle message delivery at scale.

This project demonstrates end-to-end backend engineering, distributed event pipelines, and real-world microservice system design — the exact skillset expected for **10–18 LPA backend engineering roles**.

---

##  Why This Project Stands Out

###  Fully Functional Real-Time Messaging Pipeline
```
WebSocket → Redis → Kafka → Microservices → MongoDB → WebSocket Return
```

###  Clean Microservices Architecture
Independent, isolated services with clear domain boundaries.

### Production-Grade Authentication
- JWT access & refresh tokens
- OTP verification
- Secure token lifecycle

###  Distributed Event Communication
- Redis Pub/Sub for fast fan-out
- Kafka for reliable event persistence and ordered message streams

###  Guaranteed Message Delivery
- Acknowledgements
- Retry-safe architecture

###  Complete Domain Coverage
**Auth • User • Chat • Message • Notification • WebSocket**

###  Fully Containerized Deployment
Using Docker & Docker Compose.

---

##  System Architecture
```
                   ┌───────────────────────────┐
                   │        Frontend (React)   │
                   │ - Real-time interface     │
                   │ - WebSocket client        │
                   └──────────────┬────────────┘
                                  │
                          WebSocket (Bi-directional)
                                  │
               ┌──────────────────▼──────────────────┐
               │           WebSocket-Service         │
               │ - Manages socket connections        │
               │ - Handles client events             │
               │ - Publishes to Redis + Kafka        │
               └──────────────────┬──────────────────┘
                                  │
                     Redis Pub/Sub + Kafka Stream
                                  │
      ┌─────────────┬──────────────┬──────────────┬───────────────┐
      │             │              │              │                │
 ┌────▼─────┐   ┌────▼─────┐   ┌────▼──────┐  ┌─────▼────────┐  ┌────▼───────┐
 │ Auth     │   │ User     │   │ Chat      │  │ Message       │  │ Notify     │
 │ Service  │   │ Service  │   │ Service   │  │ Service       │  │ Service    │
 └──────────┘   └──────────┘   └───────────┘  └───────────────┘  └────────────┘
```

**Database:** MongoDB  
**Communication:** REST + Event Streaming + WebSockets  
**Scalability:** Thousands of concurrent connections per node

---

##  Features

###  Auth-Service
- Register, login, logout
- OTP email verification
- JWT access & refresh tokens
- Password reset & update

###  User-Service
- Fetch user profile
- Update user details
- Delete user account

###  Chat-Service
- Create 1–1 chats
- Create and manage group chats
- Add/remove group members
- Fetch all chats for a user

###  Message-Service
- Send and receive messages
- Edit & delete messages
- Mark messages as read
- Pre-signed media upload URL generation
- Retrieve last message

###  Notification-Service
- Asynchronous notification processing
- Kafka-driven event consumption
- Fetch user notifications

###  WebSocket-Service
- Real-time bi-directional messaging
- Redis Pub/Sub broadcasting
- Kafka forwarding for persistence
- Typing indicators & online status
- Horizontal scalability (1000+ connections per node)

---

##  Tech Stack
| Layer / Module       | Technology |
|----------------------|------------|
| Backend Framework    | Go (Fiber) |
| Real-time Engine     | WebSockets |
| Streaming Layer      | Kafka      |
| Pub/Sub              | Redis      |
| Database             | MongoDB    |
| Authentication       | JWT, OTP   |
| Deployment           | Docker, Compose |
| Frontend             | React, EmailJS |

---

##  Project Structure
```
backend/
 ├── api-gateway/
 ├── services/
 │   ├── auth-service/
 │   ├── user-service/
 │   ├── chat-service/
 │   ├── message-service/
 │   ├── notification-service/
 │   └── websocket-service/
frontend/
```

Each microservice contains:
```
cmd/
internal/
configs/
keys/
.env.example
```

Follows **Clean Architecture** principles.

---

Setup & Run
1️⃣ Clone Repo
git clone https://github.com/fathimasithara01/chat-app.git
cd chat-app/backend
2️⃣ Configure Environment Files

Each service includes:

.env.example

Copy to real env:

cp .env.example .env




## ▶ Run the Project

### 1. Clone the repository
```
git clone https://github.com/fathimasithara01/chat-app.git
cd chat-app/backend
```

### 2. Configure environments
Each service includes a `.env.example` file:
```
cp .env.example .env
```

### 3. Start Backend
```
docker-compose up --build
```

### 4. Start Frontend
```
cd frontend
npm install
npm start
```

---

##  What This Project Demonstrates (For Recruiters)
- Ability to design and build **distributed systems**
- Strong understanding of **event-driven architecture**
- Hands-on experience with **Kafka, Redis, WebSockets**
- Knowledge of **scalability patterns & microservices**
- Secure **authentication and token lifecycle management**
- Dockerized deployment and **production-grade engineering**

This matches expectations for backend roles in the **10–18 LPA** range.

---

##  API Endpoints (Postman Collection)
Below is a simplified list of API endpoints grouped by microservice. These match your Postman workspace.

###  **Auth-Service**
- `POST /register`
- `POST /verifyEmail`
- `POST /login`
- `POST /request-otp`
- `POST /OTP-verify`
- `POST /refresh`
- `POST /change-password`
- `POST /logout`

###  **User-Service**
- `GET /getProfile`
- `PUT /updateProfile`
- `GET /getUserByID`
- `DELETE /deleteUser`

###  **Chat-Service**
- `POST /createChat`
- `POST /createGroup`
- `GET /listUserChats`
- `GET /getChat`
- `POST /addMember`
- `DELETE /removeMember`
- `PATCH /updateChat`

###  **Message-Service**
- `POST /sendMessage`
- `GET /listMessage`
- `POST /markRead`
- `PATCH /editMessage`
- `DELETE /deleteMessage`
- `POST /mediaUploadURL`
- `GET /lastMessage`

###  **Notification-Service**
- `POST /sendNotification`
- `GET /getUserNotification`

###  **WebSocket-Service**
- `WS /websocket`
- `WS /connect`

These endpoints are fully documented inside the included **Postman Collection**:
```
/postman/CHAT-APP.postman_collection.json
/postman/chatapp_environment.json
```
---

##  Contact
**Fathima Sithara**  
Email: `fathimasithara011@gmail.com`  
GitHub: `github.com/fathimasithara01`
