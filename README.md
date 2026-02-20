# Chat App — Real-Time Messaging Platform (Go + Microservices)

A real-time messaging platform built using Go (Fiber), WebSockets, Redis, Kafka, and MongoDB.  
This project demonstrates backend engineering skills using a microservices architecture, event-driven communication, and real-time WebSocket messaging.

---

##  Overview

This project implements a distributed messaging system with the following independent services:

- **Auth Service** — User registration, login, JWT-based authentication, refresh token handling  
- **User Service** — User profile management  
- **Chat Service** — Chat creation and group chat support  
- **Message Service** — Sending, receiving, editing, and deleting messages  
- **Notification Service** — Asynchronous user notifications  
- **WebSocket Service** — Real-time communication using WebSockets

Each service follows clean architecture principles with separated handler, service, and repository layers.

---

##  Technologies Used

| Layer              | Technology             |
| ------------------ | ---------------------- |
| Backend Framework  | Go (Fiber)             |
| Real-Time Protocol | WebSockets             |
| Event Streaming    | Kafka                  |
| Pub/Sub            | Redis                  |
| Database           | MongoDB                |
| Authentication     | JWT (Access + Refresh) |
| Deployment         | Docker, Docker Compose |


---

##  Architecture Summary

This project uses a **microservices architecture** where each service runs independently and communicates through:

- **REST APIs** — For configuration and CRUD operations  
- **WebSockets** — For real-time message delivery  
- **Redis Pub/Sub** — For low-latency event broadcasting  
- **Kafka** — For reliable event streaming and persistence  

Frontend (not included) can connect directly to WebSocket endpoints for real-time messaging.

---

##  Services

### Auth Service
- Register, login, logout
- JWT access and refresh token handling

### User Service
- Fetch profile
- Update user details
- Delete account

### Chat Service
- Create 1-on-1 chat
- Create and manage group chats
- Add/remove group members

### Message Service
- Send and receive messages
- Edit and delete messages
- Mark messages as read

### Notification Service
- Process user notifications
- Kafka-driven asynchronous event handling

### WebSocket Service
- Real-time bi-directional messaging
- Redis Pub/Sub broadcasting
- Kafka forwarding

---

##  Key Features

- WebSocket-based real-time messaging
- Independent microservices
- JWT authentication with refresh tokens
- Redis Pub/Sub for broadcasting events
- Kafka for messaging persistence
- MongoDB for storage
- Docker-based setup

---

##  Project Structure (Backend)

backend/
├── api-gateway/
├── services/
│ ├── auth-service/
│ ├── user-service/
│ ├── chat-service/
│ ├── message-service/
│ ├── notification-service/
│ └── websocket-service/


Each service has:
- `cmd/` — entry point  
- `internal/` — core logic  
- `configs/` — environment config  
- `.env.example` — environment template  

---

## Running Locally

1️ Clone the repository

  git clone https://github.com/fathimasithara01/chat-app.git
  cd chat-app/backend

2️ Configure environment

  For each service:

  cp .env.example .env

3️ Start all services

  docker-compose up --build

---

## System Flow

##  High-Level Message Flow

Client
  → WebSocket Service
    → Redis Pub/Sub (fan-out)
      → Kafka (durable stream)
        → Message Service
          → MongoDB
            → WebSocket Broadcast

---

##  Limitations

- No production deployment setup included.
- No horizontal scaling or load testing performed.
- Intended for learning and backend system design demonstration.
- Load testing pending
- Frontend not included

---

## Purpose

This project demonstrates:

- Microservices architecture design  
- Real-time system implementation  
- Event-driven backend communication  
- Concurrent handling in Go  
- Clean Architecture principles  

--- 

Author: Fathima Sithara
Role: Backend Engineer (Golang)
