# Chatroom with Golang and NATS

[![Go Version](https://img.shields.io/badge/Go-1.23-%2300BFFF.svg?logo=go&logoColor=white)](https://go.dev/)
[![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?logo=docker&logoColor=white)](https://www.docker.com/)
[![NATS](https://img.shields.io/badge/nats-%23008800.svg?logo=nats&logoColor=white)](https://nats.io/img/logo.svg)
[![Redis](https://img.shields.io/badge/redis-%23DC382D.svg?logo=redis&logoColor=white)](https://redis.io/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A scalable real-time chat application built with Go, NATS messaging system, Redis for state management, and WebSocket for client communication.

## 🚀 Features
- Real-time messaging with WebSocket support
- Room-based chat functionality with presence tracking
- Distributed message broadcasting via NATS pub/sub
- Persistent state management with Redis
- Horizontally scalable architecture
- Docker-ready deployment
- Development environment with hot-reload
- Comprehensive test coverage

## Project Structure
```bash
.
├── .air.toml.example           # Air configuration template for hot-reload
├── .gitignore                  # Git ignore rules
├── config.json.example         # Example configuration file
├── config_test.json.example    # Example test configuration
├── docker-compose.yml          # Production Docker composition
├── docker-compose-dev.yml      # Development Docker composition
├── Dockerfile                  # Multi-stage Docker build file
├── go.mod                      # Go module dependencies
├── go.sum                      # Go module checksums
|
├── api/
│   └── ws/
│       ├── handler.go          # WebSocket connection and message handling
│       └── setup.go            # WebSocket route configuration
├── cmd/
│   ├── client/
│   │   └── main.go            # CLI chat client implementation
│   └── server/
│   │   └── main.go            # Chat server entry point
├── config/
│   ├── read.go                # Configuration file reader
│   └── type.go                # Configuration type definitions
├── internal/
│   ├── app/
│   │   └── server.go          # Core application setup and lifecycle
│   ├── domain/
│   │   └── chat.go            # Chat domain types and constants
│   ├── nats/
│   │   ├── nats_client.go     # NATS client implementation
│   │   ├── publisher.go       # NATS message publishing
│   │   └── subscriber.go      # NATS subscription handling
│   └── redis/
│       └── redis_client.go    # Redis client implementation
├── pkg/
│   └── logger/
│       └── logger.go          # Structured logging package using zap
├── service/
│   └── chat_service.go        # Chat business logic implementation
└── test/
    ├── integration/
    │   └── websocket_integration_test.go  # WebSocket integration tests
    └── unit/
        ├── chat_service_test.go    # Chat service unit tests
        ├── nats_client_test.go     # NATS client unit tests
        └── redis_client_test.go    # Redis client unit tests
```

## 🏗 Architecture
The application follows clean architecture principles:

**API Layer (`api/ws`)**
- **WebSocket Handler** (`handler.go`)
  - Manages real-time WebSocket connections and client sessions
  - Handles connection lifecycle (connect, disconnect, ping/pong)
  - Validates incoming messages and routes to appropriate service methods
  - Maintains client state and room membership

- **WebSocket Setup** (`setup.go`)
  - Configures WebSocket routes and middleware
  - Initializes WebSocket upgrader with security settings
  - Sets up connection parameters and timeouts

**Service Layer (`service/chat_service.go`)**
- **Chat Service**
  - Core business logic implementation
  - Manages user sessions and room state
  - Handles message broadcasting and delivery
  - Implements chat commands (/join, /leave, /users, etc.)
  - Coordinates between WebSocket, NATS, and Redis layers

**Domain Layer (`internal/domain`)**
- **Chat Domain** (`chat.go`)
  - Defines core domain models and types
  - Contains business rules and validation logic
  - Implements chat room and user entities
  - Defines message types and formats

**Infrastructure Layer**
- **NATS Integration** (`internal/nats`)
  - Publisher: Distributes messages across server instances
  - Subscriber: Handles incoming messages from other instances
  - Manages pub/sub channels for room-based communication
- **Redis Integration** (`internal/redis`)
  - Maintains persistent state (user sessions, room info)
  - Handles distributed presence tracking
  - Manages user-room mappings and message history
- **Logging** (`pkg/logger`)
  - Structured logging using zap
  - Different log levels for development/production
  - Log file and console output support

**Application Core** (`internal/app`)
- **Server** (`server.go`)
  - Application lifecycle management
  - Component initialization and dependency injection
  - Configuration management
  - Service orchestration

**Testing** (`test`)
- **Integration Tests**
  - WebSocket communication testing
  - End-to-end functionality verification
  - Service integration validation
- **Unit Tests**
  - Individual component testing
  - Business logic validation
  - Mock-based dependency testing

**Configuration** (`config`)
- Configuration file parsing and validation
- Environment-specific settings
- Runtime configuration management

This architecture enables:
- Horizontal scalability through NATS pub/sub
- High availability with Redis state management
- Clear separation of concerns
- Easy testing and maintenance
- Flexible deployment options (local/Docker)

## 🛠 Setup & Development

### Prerequisites
- Go 1.23+
- Docker & Docker Compose

### Configuration
The application uses different configuration files depending on your setup:

1. **Docker Deployment**
```bash
# Copy the example config
cp config.json.example config.json

# Edit config.json to use container hostnames:
{
  "port": 8080,
  "nats_url": "nats://nats:4222",    # Use container hostname
  "redis_url": "redis://redis:6379",  # Use container hostname
  "log_level": "debug",
  "log_file": "server.log"
}

# Start the application
docker compose up -d --build
```

2. **Local Development**
```bash
# Copy and edit config for local development
cp config.json.example config.json

# Edit config.json to use localhost:
{
  "port": 8080,
  "nats_url": "nats://localhost:4222",  # Use localhost with exposed ports
  "redis_url": "redis://localhost:6379", # Use localhost with exposed ports
  "log_level": "debug",
  "log_file": "server.log"
}

# Start dependencies
docker compose up nats redis -d

# Run the server locally
go run cmd/server/main.go --config config.json
```

3. **Development with Hot-Reload**
```bash
# Copy configs
cp config.json.example config.json
cp .air.toml.example .air.toml

# Edit config.json as shown above for Docker setup
# Start development environment
docker compose -f docker-compose-dev.yml up --build
```


## CLI
**Running the Client**
```bash
# Run the client with default server address (localhost:8080)
go run cmd/client/main.go

# Run the client with custom server address
go run cmd/client/main.go -addr server.example.com:8080
```
| **Command**    | **Description**                              |
| -------------  |:-------------------------------------------- |
| `/help`        | Show help message and available commands     |
| `/users`       | List all active users in the system          |
| `/users <room>`| List users in a specific room                |
| `/rooms`       | List all active chat rooms                   |
| `/join <room>` | Join or switch to a specific room            |
| `/leave`       | Leave current room and return to global chat |

**Usage Examples**
```bash
# Start chatting in global room
> Hello everyone!

# Join a specific room
> /join development
[System] Joined room: development

# List room members
> /users development
[System] Users in development: alice, bob

# List all members
> /users
[System] Users: alice, bob, charlie

# Send message in current room
> Hey team!
[Sent to development] Hey team!

# Return to global chat
> /leave
[System] Returned to global chat
```
**Notes**
- New users automatically join the 'global' chat room
- Messages are only visible to users in the same room
- Room names are case-sensitive and can't have spaces
- Username is requested when starting the client

## Testing
```bash
# make config for test
cp config_test.json.example config_test.json

# Run all tests
go test -p 1 ./test/...

# Run unit tests
go test ./test/unit/...

# Run integration tests
go test ./test/integration/...
```


## Monitoring
Access logs through Dozzle at [http://localhost:9999](http://localhost:9999)  
NATS monitoring at [http://localhost:8222](http://localhost:8222)  
Redis Commander at [http://localhost:8001](http://localhost:8001) (dev environment)

**Note**: Dozzle is a lightweight container log viewer that tails our server_logs container output. It provides SQL-like analytics capabilities to query through logs, making it great for development. However, for production environments, it's recommended to use more robust solutions like:

- ELK Stack (Elasticsearch, Logstash, Kibana)
- Grafana + Promtail + Loki
- Prometheus + Grafana These solutions offer better scalability, retention policies, alerting, and production-grade monitoring capabilities.

## 🔮 Future Improvements
**Architecture & Design**
- Migrate to Hexagonal/Clean Architecture or Domain-Driven Design (DDD)
  - Better separation of domain logic from infrastructure
  - More explicit bounded contexts
  - Improved testability and maintainability
- Implement Event-Driven Architecture
  - Better handling of distributed events
  - Improved system scalability
  - Loose coupling between components

**Infrastructure**
- Enhanced Redis State Management
  - Implement state recovery mechanisms
  - Add data persistence strategies
  - Improve distributed state synchronization
  - Add cache eviction policies
- Message Queue Implementation
  - Traffic buffering
  - Message persistence for reliability
- Enhanced Load Balancing
  - Smart routing based on room capacity
  - Geographic distribution of servers

**Features**
- Extended Chat Capabilities
  - Message history
  - File sharing (Using Minio)
  - User authentication
  - Private messaging
- Enhanced Monitoring
  - Custom metrics
  - Distributed tracing
  - OpenTelemetry integration
  - End-to-end request tracking
  - Performance analytics
- Enhanced Security Measures
  - Rate limiting
  - Input validation
  - Message encryption
  - Security headers
  - CORS policy refinement

These improvements would enhance the system's reliability, maintainability, and scalability while providing a better user experience.

## 📝 License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👤 Author
Sepehr Ghafari - [GitHub](https://github.com/SphrGhfri)