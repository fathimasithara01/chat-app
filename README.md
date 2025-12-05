Chat-App — Real-Time Messaging Platform (Go + Microservices)

A production-grade real-time messaging platform built using Go (Fiber), WebSockets, Redis, Kafka, and MongoDB, structured as fully isolated microservices following clean architectural principles.

Designed for high scalability, low latency, and distributed fault tolerance, this system models how modern real-time applications such as Slack, Discord, and WhatsApp handle message delivery at scale.

This project demonstrates end-to-end backend engineering, distributed event pipelines, and real-world microservice system design — the exact skillset expected for 10–18 LPA backend engineering roles.

Why This Project Stands Out

Fully functional real-time messaging pipeline
WebSocket → Redis → Kafka → Microservices → MongoDB → WebSocket Return

Clean microservices architecture
Independent, isolated services with clear domain boundaries

Production-grade authentication
JWT access/refresh tokens, OTP verification, secure token lifecycle

Distributed event communication
Redis Pub/Sub for fast fan-out
Kafka for reliable event persistence and ordered message streams

Guaranteed message delivery
Acknowledgements and retry-safe architecture

Complete domain coverage
Auth • User • Chat • Message • Notification • WebSocket

Fully containerized deployment
Consistent environments using Docker & Docker Compose

This system demonstrates your capability to design and implement large-scale, real-time, event-driven backend systems.

System Architecture
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


Database: MongoDB
Communication: REST + Event Streaming + WebSockets
Scalability: Supports thousands of concurrent connections per node

Features
Auth-Service

Register, login, logout

OTP email verification

JWT access and refresh tokens

Password reset and update

User-Service

Fetch user profile

Update user details

Delete user account

Chat-Service

Create 1–1 chats

Create and manage group chats

Add/remove group members

Fetch all chats for a user

Message-Service

Send and receive messages

Edit and delete messages

Mark messages as read

Pre-signed media upload URL generation

Retrieve last message

Notification-Service

Asynchronous notification processing

Kafka-driven event consumption

Fetch user notifications

WebSocket-Service

Real-time bi-directional messaging

Redis Pub/Sub broadcasting

Kafka forwarding for persistence

Typing indicators, online status

Horizontal scalability (1000+ connections per node)

Tech Stack
Layer / Module	Technology
Backend Framework	Go (Fiber)
Real-time Engine	WebSockets
Streaming Layer	Kafka
Pub/Sub	Redis
Database	MongoDB
Authentication	JWT, OTP
Deployment	Docker, Compose
Frontend	React, EmailJS
Project Structure
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


Each microservice contains:

cmd/
internal/
configs/
keys/
.env


Follows Clean Architecture principles.

Run the Project
Clone the repository
git clone https://github.com/fathimasithara01/chat-app.git
cd chat-app/backend

Configure environments

Each service includes .env.example
Rename inside each service:

cp .env.example .env

Start Backend
docker-compose up --build

Start Frontend
cd frontend
npm install
npm start

What This Project Demonstrates (For Recruiters)

Ability to design and build distributed systems

Strong understanding of event-driven architecture

Hands-on experience with Kafka, Redis, WebSockets

Knowledge of scalability patterns and microservices

Secure authentication and token lifecycle management

Dockerized deployment and real production workflows

This aligns directly with expectations for backend roles in the 10–18 LPA range.

Contact

Fathima Sithara
Email: fathimasithara011@gmail.com

GitHub: github.com/fathimasithara01