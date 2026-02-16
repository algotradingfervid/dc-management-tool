# DC Management Tool

Internal web application for creating and managing Delivery Challans (DCs) across multiple projects.

## Tech Stack

- **Backend**: Go 1.26+ with Gin web framework
- **Frontend**: HTMX + Tailwind CSS
- **Database**: SQLite
- **Session Management**: SCS (alexedwards/scs)
- **CSRF Protection**: gorilla/csrf
- **Hot Reload**: Air (development)

## Quick Start

1. **Install dependencies**
   ```bash
   make setup
   ```

2. **Run development server**
   ```bash
   make dev
   ```

3. **Access the application**
   - Open browser to http://localhost:8080
   - Health check: http://localhost:8080/health

## Available Make Commands

- `make help` - Show available commands
- `make setup` - Install dependencies and set up project
- `make dev` - Run development server with hot reload
- `make build` - Build production binary
- `make run` - Build and run production binary
- `make test` - Run tests
- `make clean` - Clean build artifacts
- `make fmt` - Format Go code

## Environment Variables

Create a `.env` file (optional):

```env
APP_ENV=development
SERVER_ADDRESS=:8080
DATABASE_PATH=./data/dc_management.db
SESSION_SECRET=your-secret-key-here
UPLOAD_PATH=./static/uploads
```

## License

Internal use only - Proprietary
