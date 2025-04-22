# Scalable Music Queue System

A real-time collaborative music queue system built with Go, Redis, Kafka, and React.

## Tech Stack

### Backend
- Go (Gin framework)
- MySQL (via GORM) for persistent storage
- Redis for caching
- Kafka for event processing
- WebSockets for real-time updates

### Frontend
- React.js (Vite)
- TailwindCSS (with DaisyUI)
- WebSocket client for real-time updates

## Features
- Spotify OAuth integration
- Real-time music queue management
- Live voting system
- Room creation and management
- Collaborative playlist creation
- User profile and history

## Getting Started

### Prerequisites
- Go 1.21+
- Redis
- Kafka
- My SQL
- Node.js 18+
- Spotify Developer Account

### Environment Setup
1. Copy `.env.example` to `.env`
2. Configure your Spotify API credentials
3. Set up your database connection
4. make sure to move .env to/cmd/server near `main.go`

### Running the Backend
```bash
go mod download
go run cmd/server/main.go
```

### Running the Frontend
```bash
cd frontend
npm install
npm run dev    # start development server
```
### Building the Frontend
```bash
cd frontend
npm run build  # output to frontend/dist
```

## Architecture
- Microservices-based architecture
- Event-driven design using Kafka
- Real-time updates via WebSockets
- Redis for caching and temporary storage
- MySQL for persistent data

## License
No

