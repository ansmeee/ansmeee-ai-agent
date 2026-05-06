# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands
- `make build` — Build the server to bin/server
- `make run` — Run the server with default config (configs/config.yaml)
- `make dev` — Run the server in development mode with specific config
- `go build ./...` — Build all packages
- `go test ./...` — Run all tests
- `go test -v -run TestName ./...` — Run a specific test
- `go vet ./...` — Run vet checks
- `gofmt -w .` — Format all files
- `make test-cover` — Run tests with coverage report
- `make lint` — Run golangci-lint

## Project Architecture

This is an AI Agent framework built with Gin + LangChainGo that provides:

1. **Agent Engine** (`internal/agent/`) - Orchestrates LLM calls with tools and memory
   - `engine.go`: Core agent logic with streaming support
   - `callback.go`: Event callbacks for agent lifecycle
   - `store.go`: Agent data persistence

2. **LLM Provider** (`internal/llm/`) - LangChainGo wrapper with configuration
   - `provider.go`: OpenAI-compatible LLM client
   - `model_config_store.go`: Dynamic model configuration management

3. **Tool System** (`internal/tool/`) - Extensible tool registry
   - `registry.go`: Central tool registry
   - `calculator.go`, `datetime.go`, `web_search.go`: Built-in tools
   - Tools automatically registered and available to the LLM

4. **Memory System** (`internal/memory/`) - Session and message storage with fallback
   - `interface.go`: Memory store interface
   - `redis.go`: Redis implementation with memory fallback
   - `mysql.go`: MySQL-based session storage
   - Supports TTL and max message limits

5. **API Handlers** (`internal/handler/`) - HTTP endpoints with streaming support
   - `chat.go`: Non-streaming and streaming chat endpoints
   - `stream.go`: SSE streaming handler
   - `tool.go`: Tool discovery endpoint
   - `model_config.go`: Dynamic model configuration API

6. **Database** - GORM with MySQL master/slave support
   - Automatic connection pooling
   - Graceful degradation to single node if slave unavailable
   - Models in `internal/models/`

## Configuration

Configuration is managed through:
- YAML files in `configs/`
- Environment variables (uppercase with underscores, e.g., `LLM_API_KEY`)
- `.env` file support

Key configuration sections:
- `server`: HTTP server port and mode
- `llm`: LLM provider settings (OpenAI-compatible)
- `mysql`: Master/slave database configuration
- `redis`: Redis connection (optional, with memory fallback)
- `memory`: Session storage with TTL and limits

## Key Design Patterns

1. **Streaming Architecture**: Uses SSE for real-time responses with channel-based communication
2. **Tool Registry**: Tools implement the LangChainGo `tools.Tool` interface
3. **Memory Backends**: Pluggable storage with automatic fallback (Redis → Memory)
4. **Model Configuration**: Dynamic model switching at runtime
5. **Graceful Degradation**: Components fail gracefully (Redis unavailable → Memory fallback)

## API Endpoints

- `POST /api/v1/chat` - Non-streaming chat
- `POST /api/v1/chat/stream` - Streaming chat with SSE
- `GET /api/v1/chat/:sessionId` - Get chat history
- `DELETE /api/v1/chat/:sessionId` - Delete session
- `GET /api/v1/tools` - List available tools
- `GET /api/v1/health` - Health check

## Docker Development

- `docker-compose.yml` includes Redis service
- Build and run with `docker-compose up --build`
- Environment variables via `.env` file

## Special Notes

- Default system prompt emphasizes direct answers and tool use
- Agent has maximum 5 tool-calling iterations
- Session stores user ID for multi-user support
- Structured logging with Zap
- JWT middleware available for auth (not fully implemented)