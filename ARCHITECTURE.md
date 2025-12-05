Chat-App — System Architecture
1. Overview

Chat-App is a distributed, real-time messaging platform built using a microservices architecture.
The system is designed for low latency, high throughput, fault isolation, and horizontal scalability. Services communicate via WebSockets, Kafka, and REST. Data storage is chosen per service to match workload characteristics (MongoDB for chat/message data, PostgreSQL for auth/profile).

This document describes the high-level design, service responsibilities, event flows, data strategies, deployment considerations, and operational concerns relevant for production-ready systems and interviews.

2. High-level architecture
                   +--------------------------+
                   |       API Gateway        |
                   |     (Go Fiber + Auth)    |
                   +-----------+--------------+
                               |
        +----------------------+----------------------+
        |                      |                      |
  +-----v-----+          +-----v-----+          +-----v------+
  | Auth SRV  |          | User SRV  |          | Chat SRV   |
  | (Postgres)|          | (Postgres)|          | (MongoDB)  |
  +-----+-----+          +-----+-----+          +-----+------+
        |                        |                     |
        |                        |                     |
  +-----v-----------------------------------------------v------+
  |                    WebSocket Gateway / PubSub Layer        |
  |      - Connection handling, presence, typing, delivery     |
  |      - Publishes events into Kafka and Redis fan-out       |
  +------------------------------------------------------------+
                            |             |
                       +----v----+    +---v----+
                       | Message |    | Notify |
                       | Service |    | Service|
                       | (Mongo) |    | (Redis/|
                       +----+----+    | Mongo) |
                            |         +--------+
                            v
                         Object Storage (S3/GCS)


Infra backbone: Kafka (durable event streaming) + Redis (low-latency pub/sub & caching) + MongoDB/Postgres (storage)

3. Service responsibilities
3.1 API Gateway

Single public entry point (TLS termination, request validation).

Handles JWT validation, authentication forwarding, and central rate-limiting.

Routes to internal microservices and provides API composition where needed.

3.2 Auth Service

User registration, OTP/email verification, login/logout, token refresh, password reset.

Stores credentials and auth metadata in PostgreSQL.

Uses Redis for short-lived OTP and session caching.

3.3 User Service

Profile management, user metadata, presence status.

Stores structured profile data in PostgreSQL.

Exposes REST endpoints for profile read/write.

3.4 Chat Service

Chat and group lifecycle: create chats, create groups, add/remove members, chat metadata.

Uses MongoDB for schema-flexible chat structures and fast read of chat lists.

3.5 Message Service

High-throughput message ingestion and retrieval (send/list/edit/delete/read receipts).

Stores message documents in MongoDB designed for high write throughput and sharding.

Generates pre-signed URLs for media uploads (S3/GCS).

3.6 Notification Service

Consumes events (message.created, chat.updated) and generates push/notification delivery.

Uses Redis for caching and fan-out; persisted logs stored in MongoDB for audit/history.

3.7 WebSocket Gateway

Manages user socket connections, channel subscriptions, real-time broadcasts, typing indicators, presence.

Publishes events to Kafka for durable processing and uses Redis Pub/Sub for low-latency local fan-out.

Nodes are stateless with session routing via consistent hashing or sticky sessions when necessary.

4. Event-driven design (Kafka)

Topics

message.created — produced by WebSocket/Message service; consumed by Message and Notification services.

message.read — read receipts for analytics and sync.

chat.updated — chat metadata changes.

notification.new — downstream delivery events.

Typical flow for sending a message

Client sends message over WebSocket.

WebSocket Gateway validates and publishes message.created to Kafka.

Message Service consumes, persists to MongoDB, and produces acknowledgement/event.

Notification Service consumes and triggers push notification / in-app event.

WebSocket Gateway or consumer broadcasts to connected recipients.

This decoupling ensures non-blocking writes, retry-friendly processing, and observable event trails.

5. Data strategy & storage choices
Service	Storage	Rationale
Auth, User	PostgreSQL	ACID, relational constraints, joins for profile/auth data
Chat, Message	MongoDB	Flexible schema, document model, partitioning/sharding for write scaling
Notifications	Redis + MongoDB	Redis for fan-out and TTL data; MongoDB for durable logs
Media storage	S3 / GCS	Immutable, scalable object storage

Indexes & partitioning: Message collections must be indexed by chat_id + timestamp; use time-based partitioning or TTL for ephemeral data.

6. Request & event flows (examples)

Send message (fast path)
Client → WebSocket Gateway → message.created (Kafka) → Message Service (persist) → message.persisted (Kafka) → Notify / WebSocket broadcast

Fetch chat list
Client → API Gateway → Chat Service → MongoDB (chat metadata + last message lookup)

7. Non-functional requirements (NFRs)

Latency target: median < 50ms for in-chat message delivery to local recipients.

Durability: Message persistence with at-least-once delivery semantics; deduplication on consumer side.

Availability: 99.9% across services with multi-AZ deployment.

Scalability: Horizontal scale for WebSocket, Message, and Notification services.

Throughput: Support thousands of messages/sec via partitioned Kafka topics and sharded MongoDB.

8. Deployment & infra patterns

Containerization / Orchestration

Dockerized services.

Kubernetes preferred for production (Deployments, HPA, StatefulSets where needed).

Scaling

WebSocket Gateway: scale horizontally; use Redis for subscription propagation.

Message Service: scale based on Kafka consumer groups + MongoDB sharding.

Notification workers: autoscaled based on backlog.

CI/CD

Service-specific pipelines (lint → unit tests → integration tests → canary/rolling deploy).

Blue/green or rolling updates for zero-downtime releases.

Common infra

API Gateway (Traefik / Nginx), TLS via Let’s Encrypt or managed certs.

Observability: Prometheus + Grafana, distributed tracing (Jaeger), centralized logging (ELK/Loki).

9. Security & operational concerns

Auth & tokens

Short-lived access tokens + refresh tokens.

Token revocation list and secure refresh rotation.

Network & access

Mutual TLS between services in production where applicable.

RBAC on Kubernetes and least-privilege IAM roles for object storage.

Data protection

Encrypt at rest (DB-managed or disk-level) and in transit (TLS).

Redact sensitive logs; avoid storing secrets in repo. Use Vault or cloud-secret-manager.

Rate-limiting & abuse

Per-user and global rate-limits enforced at API Gateway.

Backpressure strategies on failure (circuit breakers, retries with backoff).

10. Monitoring & SLOs

Metrics: request latency, message processing lag (Kafka consumer lag), error rates, DB connections, node CPU/memory.

Traces: distributed tracing for message lifecycle.

Alerts: SLO breaches (latency, error rate), consumer lag thresholds, disk pressure.

On-call playbook: consumer lag alerts → scale consumers → investigate downstream sinks.

11. Folder layout (recommended)
/api-gateway
/auth-service
/user-service
/chat-service
/message-service
/notification-service
/websocket-gateway
/infra           # Terraform / k8s manifests / compose
/postman         # collections & environment
/docs            # ARCHITECTURE.md, sequence diagrams, runbooks


Keep each service self-contained with cmd/, internal/, configs/, migrations/, and a service README.

12. Production trade-offs & rationale (short)

Why Kafka + Redis? Kafka gives durable, ordered event streams suited to cross-service workflows and reprocessing. Redis provides fast fan-out for real-time broadcasts and presence cache.

Why split Chat and Message services? Separates control-plane (chat metadata) from high-throughput data-plane (messages) to scale independently.

Why MongoDB for messages? Document model matches message/thread structure and supports efficient append and read patterns; sharding supports write scale.

13. Next artifacts I can generate (if needed)

Sequence diagrams (Mermaid) for key flows.

Kafka topics & consumer group documentation.

Database schema drafts and indexes for high QPS.

Kubernetes manifests / helm charts for each service.

Runbook for operational incidents (consumer lag, DB failover).

14. Summary

This architecture balances clarity and technical depth: it demonstrates production-grade design choices while remaining readable to recruiters and hiring managers. 