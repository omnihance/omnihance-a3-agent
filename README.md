# Omnihance A3 Agent

A comprehensive web-based management platform for A3 Online MMO game servers. This application provides a beautiful, modern interface to control and manage your A3 server files, monitor system metrics, and handle user authentication.

## Overview

Omnihance A3 Agent is a full-stack application consisting of:

- **Backend**: Go-based REST API server with embedded SQLite database
- **Frontend**: ReactJS 19 web application with TypeScript, embedded in the Go binary
- **Purpose**: Provide a web interface to manage A3 Online server files, monitor system performance, and handle user access

## Features

### 🔐 Authentication & User Management

- **User Registration**: Sign up with email and password
- **User Login**: Secure authentication with bcrypt password hashing
- **Session Management**: HTTP-only cookie-based sessions with configurable timeout
- **Role-Based Access**: Support for different user roles (super admin, admin, user, viewer)
- **User Status**: Active/pending status system for new user approval
- **Auto Admin**: First registered user automatically becomes super admin

### 📁 File System Management

- **File Tree Navigation**: Browse server file system with hierarchical tree view
- **Cross-Platform Support**: Works on Windows (drive letters) and Unix-like systems
- **File Type Detection**: Automatic detection of A3-specific file types:
  - NPC files (78-byte binary files)
  - Spawn files (.n_ndt)
  - Drop files (.itm)
  - Map files (.map)
  - Text files (MIME type detection)
- **File Viewing**: View NPC files, spawn files, and text files in the browser
- **File Editing**:
  - **NPC File Editor**: Edit NPC properties including:
    - ID, Name, Respawn Rate
    - Attack configurations (3 attacks with range, area, damage)
    - Defense stats (defense, additional defense, color-based defenses)
    - Movement and attack speeds
    - HP, Level, Experience values
    - Appearance and other attributes
  - **Spawn File Editor**: Edit NPC spawn point configurations:
    - Add, remove, and modify spawn points
    - Configure NPC ID, X/Y coordinates, orientation
    - Set spawn step and other spawn properties
    - Table-based interface for managing multiple spawn points
    - **Monster Name Display**: Real-time monster name lookup based on NPC ID
    - **Map Name Display**: Shows map name in brackets when viewing spawn files (e.g., "0.n_ndt (Wolfreck)")
  - **Text File Editor**: Edit text-based configuration files
- **File Locking**: Prevents concurrent editing conflicts
- **File Revisions**: Automatic version control for all file edits
  - Revision history tracking
  - File revert functionality
  - Revision summary and count
  - Automatic backup before edits

### 📊 System Metrics & Monitoring

- **Real-Time Metrics Collection**: Automatic collection of system metrics
  - CPU usage (per-core and aggregated)
  - Memory (RAM) usage
- **Metrics Dashboard**: Visual representation of system performance
  - Metric cards showing current CPU and RAM usage
  - Interactive charts with ECharts integration
  - Time range filters (1h, 6h, 1d, 7d)
  - Smooth line charts with tooltips
- **Metrics Retention**: Configurable data retention with automatic cleanup
- **Historical Data**: Query metrics by time range for trend analysis

### 🎨 Modern Web Interface

- **Responsive Design**: Beautiful, mobile-friendly UI built with TailwindCSS
- **Dark Mode**: Theme toggle for light/dark mode support
- **Component Library**: Built with shadcn/ui components
- **Form Validation**: React Hook Form with Zod validation
- **State Management**: TanStack Query for efficient API data fetching
- **Routing**: TanStack Router for client-side routing
- **Toast Notifications**: User-friendly feedback with Sonner

### 🎮 Game Client Data Management

- **Monster Client Data**: Upload and manage monster data from A3 client files
  - Upload MON.ull files to populate monster database
  - Automatic ULL decryption and parsing
  - Bulk import with duplicate detection
  - Search and filter monsters by name
  - Real-time monster name lookup in spawn file editor
- **Map Client Data**: Upload and manage map data from A3 client files
  - Upload MC.ull files to populate map database
  - Automatic ULL decryption and parsing
  - Bulk import with duplicate detection
  - Search and filter maps by name
  - Map name display in spawn file views (extracted from filename)
- **Item Client Data**: Query item data from A3 client files
  - Search and filter items by name
  - Item data lookup support
- **Smart Data Integration**:
  - Automatic monster name resolution in spawn file editing
  - Map name extraction from spawn file filenames (e.g., "0.n_ndt" → "Wolfreck")
  - Real-time updates when editing NPC IDs

### 🔧 Additional Features

- **API Documentation**: OpenAPI/Swagger documentation embedded
- **Health Check**: `/health` endpoint for monitoring
- **CORS Support**: Configurable CORS for cross-origin requests
- **Request Logging**: Structured JSON logging with request IDs
- **Settings Management**: Key-value settings storage
- **Error Handling**: Comprehensive error codes and messages
- **Query Key Management**: Centralized React Query keys for efficient cache management

## Architecture

### Backend Structure

```
cmd/omnihance-a3-agent/
  └── main.go                    # Application entry point
  └── omnihance-a3-agent-ui/     # Frontend React application
  └── docs/                      # API documentation

internal/
  ├── config/                    # Configuration management
  ├── constants/                 # Application constants
  ├── db/                        # Database layer (SQLite)
  │   ├── users.go              # User management
  │   ├── sessions.go           # Session management
  │   ├── settings.go           # Settings storage
  │   ├── file_revisions.go     # File revision tracking
  │   ├── metrics.go            # Metrics storage
  │   ├── monster_client_data.go # Monster client data storage
  │   ├── map_client_data.go    # Map client data storage
  │   └── item_client_data.go   # Item client data storage
  ├── logger/                    # Logging abstraction
  ├── mw/                        # Middleware (auth, IP checks)
  ├── server/                    # HTTP server and routes
  │   ├── routes.go             # Route registration
  │   ├── auth_routes.go        # Authentication endpoints
  │   ├── file_system_routes.go # File operations
  │   ├── game_client_data_routes.go # Game client data endpoints
  │   ├── metrics_routes.go     # Metrics endpoints
  │   ├── session_routes.go     # Session management
  │   └── status_routes.go      # Status endpoint
  ├── services/                  # Business logic
  │   ├── file_editor_service.go
  │   ├── metrics_collector_service.go
  │   ├── collectors/           # Metric collectors (CPU, Memory)
  │   └── echarts/              # Chart generation
  └── utils/                     # Utility functions
```

### Frontend Structure

```
omnihance-a3-agent-ui/
  ├── src/
  │   ├── components/           # React components
  │   │   ├── auth-page.tsx
  │   │   ├── dashboard-layout.tsx
  │   │   ├── dashboard-page.tsx
  │   │   ├── file-tree.tsx
  │   │   ├── file-edit.tsx
  │   │   ├── file-view.tsx
  │   │   ├── npc-file-edit.tsx
  │   │   ├── npc-file-view.tsx
  │   │   ├── spawn-file-edit.tsx
  │   │   ├── spawn-file-view.tsx
  │   │   ├── text-file-edit.tsx
  │   │   ├── metric-chart.tsx
  │   │   ├── client-data-page.tsx
  │   │   ├── client-data/
  │   │   │   ├── monster-file-upload.tsx
  │   │   │   └── map-file-upload.tsx
  │   │   └── ui/              # shadcn/ui components
  │   ├── routes/              # Route definitions
  │   ├── hooks/               # Custom React hooks
  │   ├── lib/                 # Utilities and API client
  │   ├── constants.ts         # Application constants and query keys
  │   └── integrations/        # Third-party integrations
```

## Tech Stack

### Backend

- **Language**: Go 1.25
- **Web Framework**: Chi v5 (lightweight HTTP router)
- **Database**: SQLite (modernc.org/sqlite)
- **Logging**: Zerolog
- **Validation**: go-playground/validator
- **Cron**: robfig/cron/v3 (for metrics collection)
- **Crypto**: golang.org/x/crypto (bcrypt for passwords)

### Frontend

- **Framework**: React 19 with TypeScript
- **Build Tool**: Vite 7
- **Routing**: TanStack Router
- **State Management**: TanStack Query
- **Forms**: React Hook Form with Zod validation
- **Styling**: TailwindCSS 4
- **UI Components**: shadcn/ui (Radix UI primitives)
- **Charts**: ECharts with echarts-for-react
- **Icons**: Lucide React
- **Notifications**: Sonner
- **HTTP Client**: Axios

## Installation

### Prerequisites

- Go 1.25 or later
- Node.js 18+ and pnpm (for frontend development)
- Make or shell script support (for build scripts)

### Building

#### Windows

```bash
scripts\build.bat
```

#### Linux/macOS

```bash
scripts/build.sh
```

This will:

1. Build the Go backend
2. Build the React frontend
3. Embed the frontend into the Go binary

### Running

#### Windows

```bash
scripts\run.bat
```

#### Linux/macOS

```bash
scripts/run.sh
```

The application will start on `http://localhost:8080` by default.

### Development

#### Backend Development

```bash
go run cmd/omnihance-a3-agent/main.go
```

#### Frontend Development

```bash
cd cmd/omnihance-a3-agent/omnihance-a3-agent-ui
pnpm install
pnpm run dev
```

## Configuration

The application uses environment variables for configuration. A `.env` file is automatically created with default values on first run.

### Environment Variables

| Variable                              | Default                                            | Description                              |
| ------------------------------------- | -------------------------------------------------- | ---------------------------------------- |
| `PORT`                                | `8080`                                             | HTTP server port                         |
| `LOG_LEVEL`                           | `info`                                             | Logging level (debug, info, warn, error) |
| `LOG_DIR`                             | `logs`                                             | Directory for log files                  |
| `DATABASE_URL`                        | `file:omnihance-a3-agent.db?cache=shared&mode=rwc` | SQLite database connection string        |
| `METRICS_ENABLED`                     | `true`                                             | Enable/disable metrics collection        |
| `METRICS_COLLECTION_INTERVAL_SECONDS` | `60`                                               | How often to collect metrics             |
| `METRICS_RETENTION_DAYS`              | `7`                                                | How long to keep metrics data            |
| `METRICS_CLEANUP_INTERVAL_SECONDS`    | `3600`                                             | How often to clean up old metrics        |
| `REVISIONS_DIRECTORY`                 | `.revisions`                                       | Directory for file revision backups      |
| `SESSION_TIMEOUT_SECONDS`             | `2592000`                                          | Session timeout (30 days)                |
| `COOKIE_SECRET`                       | Auto-generated                                     | Secret for signing session cookies       |

## API Endpoints

### Authentication

- `POST /api/auth/sign-in` - Sign in with email and password
- `POST /api/auth/sign-up` - Register new user account

### Session

- `GET /api/session` - Get current session information
- `DELETE /api/session/sign-out` - Sign out current user
- `POST /api/session/update-password` - Update user password (requires current password, logs out all other sessions)

### Status

- `GET /api/status` - Get application status and version

### File System

- `GET /api/file-tree` - Get file tree for a path
- `GET /api/file-tree/npc-file` - Read NPC file data
- `PUT /api/file-tree/npc-file` - Update NPC file
- `GET /api/file-tree/spawn-file` - Read spawn file data
- `PUT /api/file-tree/spawn-file` - Update spawn file
- `GET /api/file-tree/text-file` - Read text file content
- `PUT /api/file-tree/text-file` - Update text file
- `POST /api/file-tree/revert-file` - Revert file to previous revision
- `GET /api/file-tree/revision-summary` - Get revision count for a file

### Metrics

- `GET /api/metrics/summary` - Get current metric values (CPU, RAM)
- `GET /api/metrics/charts` - Get metric charts with time range filter

### Game Client Data

- `GET /api/game-client-data/monsters` - Get monster client data (supports optional `s` query parameter for search)
- `POST /api/game-client-data/upload-mon-file` - Upload MON.ull file to populate monster database
- `GET /api/game-client-data/maps` - Get map client data (supports optional `s` query parameter for search)
- `POST /api/game-client-data/upload-mc-file` - Upload MC.ull file to populate map database
- `GET /api/game-client-data/items` - Get item client data (supports optional `s` query parameter for search)

### Health

- `GET /health` - Health check endpoint

## Database Schema

The application uses SQLite with the following main tables:

- **users**: User accounts with roles and status
- **sessions**: Active user sessions
- **settings**: Key-value application settings
- **file_revisions**: File edit history and revisions
- **monster_client_data**: Monster data from MON.ull files (ID, name, timestamps)
- **map_client_data**: Map data from MC.ull files (ID, name, timestamps)
- **item_client_data**: Item data from client files (ID, name, timestamps)
- **metric_names**: Metric definitions
- **metric_series**: Metric time series
- **metric_samples**: Metric data points
- **labels**: Metric labels for filtering

## Usage

1. **First Run**: Start the application and register the first user. This user will automatically become a super admin.

2. **Access the Web Interface**: Open `http://localhost:8080` in your browser.

3. **Sign In**: Use your registered credentials to sign in.

4. **Upload Game Client Data**: Navigate to the Client Data section and upload MON.ull and MC.ull files to populate the monster and map databases.

5. **Navigate Files**: Use the file tree sidebar to browse your server's file system.

6. **Edit Files**: Click on editable files (NPC files, spawn files, or text files) to view and edit them.

   - When editing spawn files, monster names are automatically displayed based on NPC ID
   - When viewing spawn files, map names are shown in brackets (e.g., "0.n_ndt (Wolfreck)")

7. **Monitor Metrics**: View system metrics on the dashboard with real-time charts.

8. **File Revisions**: All file edits are automatically backed up. Use the revision system to revert changes if needed.

## Development Commands

### Backend

- `go test ./...` - Run all tests
- `go test -v ./internal/path/to/package -run TestName` - Run specific test

### Frontend

- `pnpm run dev` - Start development server
- `pnpm run build` - Build for production
- `pnpm run lint` - Run ESLint
- `pnpm run format:write` - Format code with Prettier
- `pnpx shadcn@latest add {component-name}` - Add shadcn component

## Security Features

- Password hashing with bcrypt
- HTTP-only cookies for session management
- Signed cookies with secret key
- Input validation on all endpoints
- File path sanitization
- SQL injection prevention (parameterized queries)
- CORS configuration
- Local IP checking middleware (optional)

## License

See [LICENSE](LICENSE) file for details.

## Contributing

1. Write good readable and working code.
2. Write tests for new features.
3. Update documentation as needed.
4. Ensure all linting and formatting checks pass.

## Support

For issues, questions, or contributions, please refer to the project repository.
