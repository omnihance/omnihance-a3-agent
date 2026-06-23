# Omnihance A3 Agent

A comprehensive web-based management platform for A3 Online MMO game servers. This application provides a beautiful, modern interface to control and manage your A3 server files, monitor system metrics, run backups, and handle user authentication.

## Overview

Omnihance A3 Agent is a full-stack application consisting of:

- **Backend**: Go-based REST API server with embedded SQLite database
- **Frontend**: ReactJS 19 web application with TypeScript, embedded in the Go binary
- **Purpose**: Provide a web interface to manage A3 Online server files, monitor system performance, run file and SQL Server backups, and handle user access

## Features

### 🔐 Authentication & User Management

- **User Registration**: Sign up with email and password
  - First registered user automatically becomes super admin with active status
  - Subsequent users are created with "viewer" role and "pending" status
  - Email uniqueness validation
- **User Login**: Secure authentication with bcrypt password hashing
  - Only active users can sign in (pending, inactive, and banned users are blocked)
  - Session creation with user agent and IP address tracking
- **Session Management**: HTTP-only cookie-based sessions with configurable timeout
  - Signed cookies with secret key for security
  - Session expiration tracking
  - Password update logs out all other sessions
- **User Status Management**: Comprehensive status system for user lifecycle
  - **Pending**: New users awaiting approval (cannot sign in)
  - **Active**: Approved users who can sign in and access the system
  - **Inactive**: Temporarily disabled users (cannot sign in)
  - **Banned**: Permanently blocked users (cannot sign in)
  - Super admin users cannot have their status changed
- **User Administration** (Super Admin only):
  - List all users with pagination (default 10 per page, configurable up to 100)
  - Search users by email
  - Update user status (pending → active → inactive/banned)
  - Set/reset user passwords
  - View user roles and creation timestamps
- **Role-Based Access Control (RBAC)**: Fine-grained permission system
  - **Super Admin** (`super_admin`): Full system access
    - All permissions enabled
    - Can manage users (list, update status, set passwords)
    - Cannot have status changed by other admins
  - **Admin** (`admin`): Administrative access
    - Can view, edit, and upload files in the file browser
    - Can revert file changes
    - Can upload game client data
    - Can view metrics and game data
    - Cannot manage users
  - **Viewer** (`viewer`): Read-only access
    - Can view files
    - Can view metrics and game data
    - Cannot edit files or upload data
- **Permission Actions**:
  - `view_files`: View file system and file contents (super_admin, admin, viewer)
  - `download_files`: Create and use one-day user-bound download links for file-browser and backup output files through `POST /api/file-tree/download-link` and `GET /api/file-tree/download/{token}` (super_admin, admin)
  - `edit_files`: Edit files and upload files through the file browser (super_admin, admin)
  - `revert_files`: Revert files to previous revisions (super_admin, admin)
  - `upload_game_data`: Upload MON.ull and MC.ull files (super_admin, admin)
  - `manage_users`: Manage user accounts (super_admin only)
  - `manage_server`: Manage server processes and startup sequence (super_admin, admin)
  - `view_metrics`: View system metrics dashboard (super_admin, admin, viewer)
  - `view_game_data`: View monster, map, and item data (super_admin, admin, viewer)

### 📁 File System Management

- **File Tree Navigation**: Browse server file system with hierarchical tree view
- **Cross-Platform Support**: Works on Windows (drive letters) and Unix-like systems
- **File Type Detection**: Automatic detection of A3-specific file types:
  - NPC files (78-byte binary files)
  - Quest files (.dat)
  - Spawn files (.n_ndt)
  - Drop files (.itm)
  - Item combination data files
  - Map files (.map)
  - Text files (MIME type detection)
- **File Viewing**: View NPC files, quest files, spawn files, drop files, item files, item combination data files, and text files in the browser
- **File Duplication**: Right-click any file in the file explorer to duplicate it with a custom name
- **Directory Downloads**: Right-click a directory to request a ZIP download. The directory is compressed in the background through a tagged one-time backup job, stored under `DIRECTORY_DOWNLOADS_DIRECTORY`, and delivered through the same secure temp-link download flow used for files and backup outputs.
- **Chunked File Uploads**: Drag files or folders into a real directory, or use the Upload button next to Show dotfiles to open a drop zone with file and folder pickers. Uploads are disabled on the root drive-listing page, run one file at a time per browser tab, auto-rename conflicts with the existing `(copy)` naming pattern, retry transient chunk failures, verify SHA-256 before finalizing, and keep abandoned temp data hidden until server cleanup removes it.
- **File Editing**:
  - **NPC File Editor**: Edit NPC properties including:
    - ID, Name, Respawn Rate
    - Attack configurations (3 attacks with range, area, damage)
    - Defense stats (defense, additional defense, color-based defenses)
    - Movement and attack speeds
    - HP, Level, Experience values
    - Appearance and other attributes
  - **Quest File Editor**: Edit A3 quest file configurations:
    - Quest header configuration (Quest ID, Giver NPC, Target NPC, Min/Max Level, Flags)
    - Reward configuration with add/remove controls for 3 item reward slots, plus Experience, Woonz, and Lore rewards
    - Type-driven objective management for 7 objective slots:
      - KILL, QUESTITEM, BRINGNPC, DROP, FIND, and UNUSED objective types
      - Objective type selection enables only the fields used by that type
      - Add/remove controls for unused objective slots, drop item slots, and quest item requirements
      - Target monster/NPC IDs, kill counts, quest item IDs, required item counts, drop item probabilities, and optional DROP/FIND names
    - Continuation quest configuration with add/remove controls for 3 chain quest IDs
    - Form-based interface with validation
    - Binary-safe save behavior that preserves existing padding and unknown bytes while writing proper A3 sentinel values for removed slots
  - **Spawn File Editor**: Edit NPC spawn point configurations:
    - Add, remove, and modify spawn points
    - Configure NPC ID, X/Y coordinates, orientation
    - Set spawn step and other spawn properties
    - Table-based interface for managing multiple spawn points
    - **Monster Name Display**: Real-time monster name lookup based on NPC ID
    - **Map Name Display**: Shows map name in brackets when viewing spawn files (e.g., "0.n_ndt (Wolfreck)")
  - **Drop File Editor**: Edit monster drop file configurations:
    - Add, remove, and modify drop entries
    - Configure item code, drop rate, and drop group values
    - Choose uploaded item client data from a dropdown or enter custom item codes manually
    - **Item Name Display**: Shows item names when IT0-IT3 client data has been uploaded
  - **Item File Editor**: View and edit A3 item files named `0`, `0ex`, `1`, `2`, and `3`:
    - Edit known item fields while preserving hidden unknown bytes
    - Decode and save item names with selectable UTF-8, EUC-KR, GBK, Big5, or Shift-JIS encoding for legacy Chinese, Korean, and Japanese item files
    - Keep row, type, item code base, and computed item code read-only
    - Edit IT0 equipment levels, IT1 option fields, IT2 skill/class fields, and IT3 price/name fields
    - View and edit `0ex` level extensions only when sibling `0` exists in the same directory
    - Add or delete `0ex` extension rows while preventing duplicate base item rows
  - **Item Combination Data Editor**: Edit craft formula data:
    - Add, remove, and modify formulas
    - Configure 10 ingredient item codes, success rate, and outcome item code
    - Search formulas by outcome code or uploaded item name
    - Preserve unknown bytes for existing rows while saving edited formula fields
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
  - Global dashboard time window selector (default: last hour)
  - Preset ranges: 1h, 6h, 1d, 7d
  - Adaptive server-side aggregation to prevent overcrowded x-axis labels
  - Extensible time-window model ready for future custom from/to filtering
  - Smooth line charts with tooltips
- **Release Update Banner**: Dashboard notification when a newer stable GitHub release is available
  - Checks the latest GitHub release in the background, not on every dashboard refresh
  - Uses semantic version comparison with support for `v1.2.3` and `1.2.3` tags
  - Shows the latest available release for `dev` builds
  - Configurable check interval with a safe default of 1 hour
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
- **Item Client Data**: Upload and query item data from A3 client files
  - Upload IT0.ull, IT1.ull, IT2.ull, and IT3.ull files to populate item data
  - Automatic ULL decryption and parsing for each item file type
  - Bulk import with duplicate detection per item type
  - Search and filter items by name
  - Item data lookup support
- **Client Data Counts**:
  - View imported record counts for monsters, maps, and each item file type
  - Use counts to confirm which client datasets have already been uploaded
- **Smart Data Integration**:
  - Automatic monster name resolution in spawn file editing
  - Map name extraction from spawn file filenames (e.g., "0.n_ndt" → "Wolfreck")
  - Real-time updates when editing NPC IDs

### Server View

- **High-level server overview**:
  - Shows Main, Account, Zone, and Battle server cards from each server's `SvrInfo.ini`
  - Missing server paths stay visible as "Missing Path" cards so users know what to configure
  - Uses `MAIN_SERVER_PATH`, `ACCOUNT_SERVER_PATH`, `ZONE_SERVER_PATH`, and `BATTLE_SERVER_PATH` from Settings
- **Sync from raw server files**:
  - Syncs Main and Zone `MapInfo.ini` map-to-zone data
  - Syncs raw monster spawn rows from Zone Server `ZoneData/map/*.n_ndt`
  - Syncs raw monster drop rows from Zone Server `ZoneData/npc/*.itm`
  - Syncs raw shop rows from Zone Server `ZoneData/shop/*.txt`
  - Syncs game master rows from Zone Server `GMInfo.ini`
  - Continues syncing when individual files fail and records warnings for skipped files
  - Uses a sync lock so only one Server View sync can run at a time
- **Server View pages**:
  - Main Server Map Info shows map name, map ID, and zone ID
  - Zone Loaded Maps shows maps loaded by the Zone Server
  - Monster Spawns starts map-first with map or monster search, then shows monster counts per map
  - Monster Drops supports universal NPC, item name, and item ID search, with per-NPC drop details
  - Shops supports NPC and item search, with per-NPC shop item details
  - Uploaded client data is used for `Name (ID)` labels for maps, monsters, and items

### 🚀 Server Process Management

- **Sequential Server Startup/Shutdown**: Manage complex multi-process server startup sequences
  - Configure multiple executables and batch files in a specific startup order
  - Sequential startup with health checks (waits for each process to be ready before starting the next)
  - Reverse-order shutdown for clean server stops
  - Support for executables (.exe) and batch files (.bat, .cmd)
- **Process Configuration**:
  - Add processes via file tree context menu (right-click on .exe/.bat/.cmd files) or manage server page
  - Friendly names for easy identification
  - Optional port configuration for health verification
  - Path validation (ensures file exists and is valid executable/batch file)
  - Duplicate path prevention
  - Drag-and-drop reordering of startup sequence
- **Process Monitoring**:
  - Real-time status display (Running/Stopped)
  - Port status checking (if configured)
  - Uptime tracking (current uptime for running processes, last uptime for stopped processes)
  - Start/end time recording
  - Automatic status polling when processes are running
- **Individual Process Control**:
  - Start/stop individual processes
  - Start/stop entire server sequence
  - Health check verification (port check if available, process check otherwise)
  - Timeout handling (60 seconds per process)
- **Access Control**:
  - Admin and Super Admin: Full management (add, edit, delete, start, stop, reorder)
  - Viewer: Read-only access (can view process status and uptime, cannot manage)

### 🗄️ SQL Server ODBC User DSN Management

- **32-bit User DSN management** for legacy SQL Server ODBC driver (`SQL Server`)
- **Full CRUD support**:
  - List SQL Server DSNs
  - View DSN details
  - Create and update DSN settings
  - Delete DSNs
- **Default A3 DSN creation**:
  - Create the common A3 SQL Server User DSNs from a shared server name and login ID
  - Creates missing defaults and skips existing DSNs without overwriting them
  - Default DSNs: `A3Friend`, `A3ItemEvent`, `A3RcvResult`, `A3SerialList`, `ASD`, `EventA3`, `FriendDB`, `HSDB`, `LETTERDB`, `LocalServer`, `Login202`, `NEWASD`
- **Connection testing** before save:
  - Test SQL auth/server/database connectivity using submitted form values
- **Registry parity** with Windows ODBC UI:
  - `HKCU\Software\WOW6432Node\ODBC\ODBC.INI\ODBC Data Sources`
  - `HKCU\Software\WOW6432Node\ODBC\ODBC.INI\<dsnName>`
  - Persists `Driver`, `Server`, `Database`, and `LastUser`
  - Does not persist SQL passwords in the registry

### 🔧 Additional Features

#### Directory Shortcuts and Settings

- **Directory shortcuts**:
  - Pin frequently used directories in the file browser per user
  - Prevent duplicate shortcuts through normalized path checks
  - Reject root-directory shortcuts to keep navigation focused
  - Limit shortcut count with `DIRECTORY_SHORTCUTS_LIMIT`
- **Game server settings**:
  - Manage `DB_HOST`, `DB_PORT`, `DB_USER`, and `DB_PASS` from the Settings page
  - Manage `ZONE_SERVER_PATH`, `ACCOUNT_SERVER_PATH`, `MAIN_SERVER_PATH`, and `BATTLE_SERVER_PATH`
  - Server path settings use searchable directory suggestions and must point to directories containing `SvrInfo.ini`
  - Validate supported keys and value types before save
  - Reuse saved SQL Server values as backup defaults

#### Backup Jobs

- **File and directory backups**:
  - Create manual or scheduled jobs for a selected file or directory
  - Destination directories are created when missing, and destination paths must be directories
  - Source path and destination path fields support searchable path suggestions
- **Directory download jobs**:
  - Directory-download requests create tagged no-repeat file backup jobs with the `directory_download` tag
  - Only one directory-download compression job can run at a time; repeat requests for the same running directory resume polling, while requests for a different directory return a conflict
  - Completed directory ZIPs are reused until the source directory fingerprint changes
  - Directory-download backup jobs hide their destination in the Backups UI and show the run trigger as `Directory Download`
- **Local SQL Server backups**:
  - Back up one or more comma-separated database names
  - SQL Server host, port, username, and password can be prefilled from settings
  - Remote SQL Server hosts are rejected; only local SQL Server backups are supported
  - Each database is backed up to `.bak` first, then archived for download
- **Scheduling and control**:
  - Optional cron expressions enable automatic runs; empty cron values are manual-only
  - Jobs can be active, inactive, running, deleted, or future extensible statuses
  - Running jobs cannot be edited or deleted
  - Manual runs, cancellation, and paginated run history are available from the Backups page
  - Cron collisions create skipped run records instead of starting concurrent runs
- **Backup outputs**:
  - Archives are ZIP files using best Deflate compression
  - Archive passwords use AES-256 ZIP encryption when provided
  - Output files use `{item-name}-{job-id}-YYYYMMDDHHMMSS.zip`; a numeric suffix is added if needed
  - Run details record logs, errors, cancellation state, and output file links

#### Platform Features

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
  │   ├── directory_shortcuts.go # Per-user pinned directories
  │   ├── file_revisions.go     # File revision tracking
  │   ├── metrics.go            # Metrics storage
  │   ├── monster_client_data.go # Monster client data storage
  │   ├── map_client_data.go    # Map client data storage
  │   ├── item_client_data.go   # Item client data storage
  │   └── server_view.go        # Synced server overview data storage
  ├── logger/                    # Logging abstraction
  ├── mw/                        # Middleware (auth, IP checks)
  ├── permissions/               # RBAC permission system
  │   └── permissions.go        # Permission definitions and checks
  ├── server/                    # HTTP server and routes
  │   ├── routes.go             # Route registration
  │   ├── auth_routes.go        # Authentication endpoints
  │   ├── users_routes.go       # User management endpoints
  │   ├── file_system_routes.go # File operations
  │   ├── game_client_data_routes.go # Game client data endpoints
  │   ├── metrics_routes.go     # Metrics endpoints
  │   ├── session_routes.go     # Session management
  │   ├── server_routes.go      # Server process management endpoints
  │   ├── directory_shortcuts_routes.go # Directory shortcut endpoints
  │   ├── settings_routes.go    # Game server settings endpoints
  │   ├── server_view_routes.go # Server View sync and query endpoints
  │   ├── sqlserver_odbc_routes.go # SQL Server ODBC DSN endpoints
  │   ├── backup_routes.go      # Backup job and run endpoints
  │   ├── permissions.go        # Permission checking utilities
  │   └── status_routes.go      # Status endpoint
  ├── services/                  # Business logic
  │   ├── file_editor_service.go
  │   ├── metrics_collector_service.go
  │   ├── process_service.go    # Process management (start, stop, health checks)
  │   ├── server_view_service.go # Server View file sync and aggregation
  │   ├── server_manager_service.go # Server sequence orchestration
  │   ├── collectors/           # Metric collectors (CPU, Memory)
  │   └── echarts/              # Chart generation
  └── utils/                     # Utility functions
    └── port_checker.go          # TCP port availability checking
```

Backup-related backend code lives in `internal/db/backup_jobs.go`, `internal/server/backup_routes.go`, `internal/services/backup_service.go`, and `internal/services/backup_sql_service_*.go`.

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
  │   │   ├── quest-file-edit.tsx
  │   │   ├── quest-file-view.tsx
  │   │   ├── spawn-file-edit.tsx
  │   │   ├── spawn-file-view.tsx
  │   │   ├── item-combination-data-file-edit.tsx
  │   │   ├── item-combination-data-file-view.tsx
  │   │   ├── text-file-edit.tsx
  │   │   ├── metric-chart.tsx
  │   │   ├── client-data-page.tsx
  │   │   ├── manage-server-page.tsx
  │   │   ├── settings-page.tsx
  │   │   ├── sql-server-odbc-page.tsx
  │   │   ├── backup-page.tsx
  │   │   ├── client-data/
  │   │   │   ├── monster-file-upload.tsx
  │   │   │   ├── map-file-upload.tsx
  │   │   │   └── item-file-upload.tsx
  │   │   └── ui/              # shadcn/ui components
  │   ├── routes/              # Route definitions
  │   │   ├── manage-server.tsx
  │   │   ├── settings.tsx
  │   │   ├── sql-server-odbc.tsx
  │   │   └── backups.tsx
  │   ├── hooks/               # Custom React hooks
  │   │   └── use-permissions.ts
  │   ├── lib/                 # Utilities and API client
  │   ├── constants.ts         # Application constants and query keys
  │   └── integrations/        # Third-party integrations
```

The Backups, Settings, and SQL Server ODBC pages are implemented in `src/components/backup-page.tsx`, `src/components/settings-page.tsx`, and `src/components/sql-server-odbc-page.tsx` with matching routes under `src/routes/`.

## Tech Stack

### Backend

- **Language**: Go 1.25
- **Web Framework**: Chi v5 (lightweight HTTP router)
- **Database**: SQLite (modernc.org/sqlite)
- **Logging**: Zerolog
- **Validation**: go-playground/validator
- **Cron**: robfig/cron/v3 (for metrics collection and backup scheduling)
- **Archives**: yeka/zip for ZIP creation and AES-encrypted backup archives
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

| Variable                              | Default                                            | Description                                                  |
| ------------------------------------- | -------------------------------------------------- | ------------------------------------------------------------ |
| `PORT`                                | `8080`                                             | HTTP server port                                             |
| `LOG_LEVEL`                           | `info`                                             | Logging level (debug, info, warn, error)                     |
| `LOG_DIR`                             | `logs`                                             | Directory for log files                                      |
| `DATABASE_URL`                        | `file:omnihance-a3-agent.db?cache=shared&mode=rwc` | SQLite database connection string                            |
| `METRICS_ENABLED`                     | `true`                                             | Enable/disable metrics collection                            |
| `METRICS_COLLECTION_INTERVAL_SECONDS` | `60`                                               | How often to collect metrics                                 |
| `METRICS_RETENTION_DAYS`              | `7`                                                | How long to keep metrics data                                |
| `METRICS_CLEANUP_INTERVAL_SECONDS`    | `3600`                                             | How often to clean up old metrics                            |
| `VERSION_CHECK_INTERVAL_SECONDS`      | `3600`                                             | How often to check GitHub for new releases                   |
| `REVISIONS_DIRECTORY`                 | `.revisions`                                       | Directory for file revision backups                          |
| `BACKUPS_DIRECTORY`                   | `.backups`                                         | Directory for internal backup lock files                     |
| `DIRECTORY_DOWNLOADS_DIRECTORY`       | `.directory-download`                              | Directory for generated directory-download ZIP archives      |
| `MAX_FILE_UPLOAD_SIZE_MB`             | `2`                                                | Maximum multipart upload size in MB                          |
| `DIRECTORY_SHORTCUTS_LIMIT`           | `5`                                                | Maximum pinned directories per user (`0` disables the limit) |
| `RUNNING_IN_DOCKER`                   | `false`                                            | Disable host metrics collection when running in Docker       |
| `SESSION_TIMEOUT_SECONDS`             | `2592000`                                          | Session timeout (30 days)                                    |
| `COOKIE_SECRET`                       | Auto-generated                                     | Secret for signing session cookies                           |

### Version Checks

The agent checks `https://api.github.com/repos/omnihance/omnihance-a3-agent/releases/latest` in the background and caches the result in memory. The dashboard and `/api/status` read the cached value, so refreshing the UI does not call GitHub.

Only stable GitHub releases are considered because GitHub's latest release endpoint excludes draft and prerelease releases. If the running version is `dev`, the dashboard shows the latest available release. For release builds, the checker compares semantic versions and shows the banner only when the GitHub release is newer.

## API Endpoints

### Authentication

- `POST /api/auth/sign-in` - Sign in with email and password
- `POST /api/auth/sign-up` - Register new user account

### Session

- `GET /api/session` - Get current session information
- `DELETE /api/session/sign-out` - Sign out current user
- `POST /api/session/update-password` - Update user password (requires current password, logs out all other sessions)

### User Management

- `GET /api/users` - List users with pagination and search (requires `manage_users` permission)
  - Query parameters: `page` (default: 1), `pageSize` (default: 10, max: 100), `s` (search by email)
- `GET /api/users/statuses` - Get available user statuses (requires `manage_users` permission)
- `PATCH /api/users/{id}/status` - Update user status (requires `manage_users` permission)
  - Cannot update super admin status
  - New status must be different from current status
- `PATCH /api/users/{id}/password` - Set user password (requires `manage_users` permission)
  - Password must be at least 6 characters

### Status

- `GET /api/status` - Get application status, current version, cached latest release details, and update availability

### File System

- `GET /api/file-tree` - Get file tree for a path
- `GET /api/file-tree/npc-file` - Read NPC file data
- `PUT /api/file-tree/npc-file` - Update NPC file
- `GET /api/file-tree/quest-file` - Read quest file data
- `PUT /api/file-tree/quest-file` - Update quest file
- `GET /api/file-tree/spawn-file` - Read spawn file data
- `PUT /api/file-tree/spawn-file` - Update spawn file
- `GET /api/file-tree/text-file` - Read text file content
- `PUT /api/file-tree/text-file` - Update text file
- `GET /api/file-tree/drop-file` - Read A3 drop file data
- `PUT /api/file-tree/drop-file` - Update A3 drop file data
- `GET /api/file-tree/item-file` - Read A3 item file data for files named `0`, `0ex`, `1`, `2`, or `3`; accepts optional `name_encoding` (`utf-8`, `euc-kr`, `gbk`, `big5`, `shift-jis`)
- `PUT /api/file-tree/item-file` - Update A3 item file data while preserving hidden unknown fields and saving item names using the request `name_encoding`
- `GET /api/file-tree/item-combination-data` - Read A3 item combination data
- `PUT /api/file-tree/item-combination-data` - Update A3 item combination data
- `POST /api/file-tree/revert-file` - Revert file to previous revision
- `POST /api/file-tree/duplicate-file` - Duplicate a file in the same directory
- `POST /api/file-tree/uploads` - Start a file-browser upload batch for a non-root destination directory and reserve conflict-safe target names
- `PUT /api/file-tree/uploads/{upload_id}/files/{file_id}/chunks/{chunk_index}` - Upload one binary file chunk with retry support
- `POST /api/file-tree/uploads/{upload_id}/files/{file_id}/complete` - Verify SHA-256 and finalize one uploaded file
- `POST /api/file-tree/uploads/{upload_id}/heartbeat` - Keep an active upload session alive while the browser tab is open
- `DELETE /api/file-tree/uploads/{upload_id}` - Cancel an upload session and remove hidden temp data
- `GET /api/file-tree/revision-summary` - Get revision count for a file
- `POST /api/file-tree/download-link` - Create or reuse a one-day user-bound download link for a file (requires `download_files` permission)
- `POST /api/file-tree/directory-download-link` - Create, reuse, or start a background directory ZIP job and return either a secure download URL or polling metadata (requires `download_files` permission)
- `GET /api/file-tree/directory-downloads/{run_id}` - Poll a directory-download run until it returns a secure download URL or terminal status
- `GET /api/file-tree/download/{token}` - Download a file through a valid user-bound temp link and record the download event

### Metrics

- `GET /api/metrics/summary` - Get current metric values (CPU, RAM)
- `GET /api/metrics/charts` - Get metric charts with time window filtering and adaptive aggregation
  - Query parameters: `range` (default `1h`, supports `1h`, `6h`, `1d`, `7d`), optional `from` and `to` (Unix seconds, must be provided together)

### Game Client Data

- `GET /api/game-client-data/counts` - Get imported record counts for monsters, maps, and item file types
- `GET /api/game-client-data/monsters` - Get monster client data (supports optional `s` query parameter for search)
- `POST /api/game-client-data/upload-mon-file` - Upload MON.ull file to populate monster database
- `GET /api/game-client-data/maps` - Get map client data (supports optional `s` query parameter for search)
- `POST /api/game-client-data/upload-mc-file` - Upload MC.ull file to populate map database
- `GET /api/game-client-data/items` - Get item client data (supports optional `s` query parameter for search)
- `POST /api/game-client-data/upload-it0-file` - Upload IT0.ull file to populate item data
- `POST /api/game-client-data/upload-it1-file` - Upload IT1.ull file to populate item data
- `POST /api/game-client-data/upload-it2-file` - Upload IT2.ull file to populate item data
- `POST /api/game-client-data/upload-it3-file` - Upload IT3.ull file to populate item data

### Directory Shortcuts

- `GET /api/directory-shortcuts` - List current user's pinned directory shortcuts with limit metadata
- `POST /api/directory-shortcuts` - Create a pinned directory shortcut for the current user
- `DELETE /api/directory-shortcuts/{id}` - Delete one of the current user's pinned directory shortcuts

### Settings

- `GET /api/settings` - List supported game server settings and definitions (requires `manage_server` permission)
- `POST /api/settings` - Create a supported setting (requires `manage_server` permission)
- `PUT /api/settings/{key}` - Update a supported setting (requires `manage_server` permission)
- `DELETE /api/settings/{key}` - Delete a supported setting (requires `manage_server` permission)
  - Supported keys: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, `ZONE_SERVER_PATH`, `ACCOUNT_SERVER_PATH`, `MAIN_SERVER_PATH`, and `BATTLE_SERVER_PATH`
  - Server path settings must point to directories containing `SvrInfo.ini`

### Server View

- `GET /api/server-view` - Get Server View overview cards, sync status, missing settings, and recent warnings (requires `view_game_data` permission)
- `GET /api/server-view/sync/status` - Get current and latest Server View sync status (requires `view_game_data` permission)
- `POST /api/server-view/sync` - Start a background Server View sync (requires `manage_server` permission; returns conflict if a sync is already running)
- `GET /api/server-view/main/maps` - List Main Server map-to-zone rows (supports optional `q` search)
- `GET /api/server-view/zone/maps` - List Zone Server loaded map-to-zone rows (supports optional `q` search)
- `GET /api/server-view/zone/spawns` - List aggregated monster spawn rows (supports optional `map_q` and `npc_q` search)
- `GET /api/server-view/zone/drops` - List NPCs with drop row counts (supports optional `q` search by NPC, item name, or item ID)
- `GET /api/server-view/zone/drops/{npc_id}` - List drop details for one NPC (supports optional `q` item search)
- `GET /api/server-view/zone/shops` - List NPC shops with item counts (supports optional `q` search by NPC or item)
- `GET /api/server-view/zone/shops/{npc_id}` - List shop item details for one NPC (supports optional `q` item search)

### Server Management

- `GET /api/server/processes` - List all server processes (ordered by sequence)
- `POST /api/server/processes` - Create a new server process (requires `manage_server` permission)
- `GET /api/server/processes/{id}` - Get a specific server process
- `PUT /api/server/processes/{id}` - Update a server process (requires `manage_server` permission)
- `DELETE /api/server/processes/{id}` - Delete a server process (requires `manage_server` permission)
- `POST /api/server/processes/reorder` - Reorder server processes (requires `manage_server` permission)
- `POST /api/server/start` - Start full server sequence (requires `manage_server` permission)
- `POST /api/server/stop` - Stop full server sequence (requires `manage_server` permission)
- `POST /api/server/processes/{id}/start` - Start an individual process (requires `manage_server` permission)
- `POST /api/server/processes/{id}/stop` - Stop an individual process (requires `manage_server` permission)
- `GET /api/server/processes/{id}/status` - Get process status (running, port status, uptime)

### SQL Server ODBC

- `GET /api/odbc/sqlserver-dsns` - List SQL Server User DSNs (requires `manage_server` permission)
- `GET /api/odbc/sqlserver-dsns/{name}` - Get SQL Server User DSN details (requires `manage_server` permission)
- `POST /api/odbc/sqlserver-dsns` - Create SQL Server User DSN (requires `manage_server` permission)
- `POST /api/odbc/sqlserver-dsns/defaults` - Create missing default A3 SQL Server User DSNs and skip existing DSNs (requires `manage_server` permission)
- `PUT /api/odbc/sqlserver-dsns/{name}` - Update SQL Server User DSN (requires `manage_server` permission)
- `DELETE /api/odbc/sqlserver-dsns/{name}` - Delete SQL Server User DSN (requires `manage_server` permission)
- `POST /api/odbc/sqlserver-dsns/test` - Test SQL Server connection using request payload (requires `manage_server` permission)

### Backups

- `GET /api/backups/jobs` - List backup jobs (requires `manage_server` permission)
- `POST /api/backups/jobs` - Create a file or local SQL Server backup job (requires `manage_server` permission)
- `GET /api/backups/jobs/{id}` - Get backup job details, including stored passwords for authorized users
- `PUT /api/backups/jobs/{id}` - Update a backup job and reschedule cron if needed
- `DELETE /api/backups/jobs/{id}` - Soft delete a backup job; running jobs cannot be deleted
- `POST /api/backups/jobs/{id}/run` - Manually trigger a backup job
- `POST /api/backups/jobs/{id}/cancel` - Cancel a running backup job
- `GET /api/backups/jobs/{id}/runs` - List paginated run history for a backup job
- `GET /api/backups/runs/{run_id}` - Get run details, logs, errors, and output file metadata
- `POST /api/backups/runs/{run_id}/files/{file_id}/download-link` - Create or reuse a one-day user-bound download link for a successful backup output file (requires `download_files` permission)
- `GET /api/backups/runs/{run_id}/files/{file_id}/download` - Compatibility route that redirects successful backup output downloads through the temp-link flow
- `GET /api/backups/path-search` - Search local source or destination paths
- `GET /api/backups/defaults/sql-server` - Get SQL Server backup defaults and local server status

### Health

- `GET /health` - Health check endpoint

## Database Schema

The application uses SQLite with the following main tables:

- **users**: User accounts with roles and status
- **sessions**: Active user sessions
- **settings**: Key-value application settings
- **directory_shortcuts**: Per-user pinned directories for faster file navigation
- **file_revisions**: File edit history and revisions
- **monster_client_data**: Monster data from MON.ull files (ID, name, timestamps)
- **map_client_data**: Map data from MC.ull files (ID, name, timestamps)
- **item_client_data**: Item data from client files (ID, name, timestamps)
- **server_processes**: Server process configurations
  - Stores process name, file path, optional port, sequence order
  - Tracks start/end times for uptime calculation
  - Enforces unique paths to prevent duplicates
- **backup_jobs**: Backup job definitions, scheduling metadata, paths, SQL settings, statuses, and optional tags such as `directory_download`
- **backup_runs**: Backup run history, trigger type, status, output logs, errors, and cancellation timestamps
- **backup_run_files**: Output archive files created by each backup run
- **directory_download_archives**: Cached generated directory ZIP archives keyed by normalized directory path and recursive fingerprint
- **file_download_links**: One-day user-bound file download links, file fingerprints, source context, and per-link download counts
- **file_download_events**: Per-request file download audit events with user, link, source context, file fingerprint, IP address, and user agent
- **server_view_svr_info_rows**: Raw `SvrInfo.ini` rows for Main, Account, Zone, and Battle servers
- **server_view_map_zones**: Raw Main and Zone Server map-to-zone references
- **server_view_spawn_rows**: Raw monster spawn rows from zone map `.n_ndt` files
- **server_view_drop_rows**: Raw monster drop rows from zone NPC `.itm` files
- **server_view_shop_rows**: Raw NPC shop rows from zone shop `.txt` files
- **server_view_game_master_rows**: Game master names and levels from Zone Server `GMInfo.ini`
- **server_view_sync_runs**: Server View sync run status, timestamps, lock state, and errors
- **server_view_sync_warnings**: Per-run warnings for skipped files or parser failures
- **metric_names**: Metric definitions
- **metric_series**: Metric time series
- **metric_samples**: Metric data points
- **labels**: Metric labels for filtering

## Usage

1. **First Run**: Start the application and register the first user. This user will automatically become a super admin with active status.

2. **Access the Web Interface**: Open `http://localhost:8080` in your browser.

3. **Sign In**: Use your registered credentials to sign in. Only users with "active" status can sign in.

4. **User Management** (Super Admin only):
   - Navigate to the Users page to manage user accounts
   - View all registered users with pagination and search
   - Update user status (approve pending users, deactivate, or ban users)
   - Set/reset user passwords
   - Note: Super admin users cannot have their status changed

5. **Upload Game Client Data**: Navigate to the Client Data section and upload MON.ull, MC.ull, and IT0.ull through IT3.ull files to populate monster, map, and item databases (requires admin or super admin role).

6. **Navigate Files**: Use the file tree sidebar to browse your server's file system (all authenticated users can view). Pin frequently used directories as shortcuts, right-click files to duplicate them, and download files or directories with admin or super admin access. Admins and super admins can upload files or folders inside a selected directory by dragging into the browser or using the Upload button beside Show dotfiles; uploads are queued per tab, conflicts are auto-renamed, and progress stays visible in the bottom-right panel. Directory downloads compress in the background; keep the page open and do not refresh so the download can start automatically when ready. If you refresh, click the same directory download again to resume polling for the in-progress job.

7. **Edit Files**: Click on editable files (NPC files, quest files, spawn files, drop files (monster drop configurations), item files, item combination data files, or text files) to view and edit them (requires admin or super admin role).
   - **Quest Files**: Edit quest configurations with type-aware objectives, add/remove controls for optional slots, and binary-safe padding preservation
   - **Drop File Editor**: Edit drop files (monster drop configurations) by adding, removing, and modifying entries, selecting item codes, and displaying item names when IT0-IT3 client data has been uploaded
   - **Item File Editor**: Edit item files named `0`, `1`, `2`, and `3` in place; edit `0ex` only when sibling `0` exists, with add/delete controls for extension rows and selectable legacy name encoding for Chinese/Korean/Japanese item files
   - **Item Combination Data Editor**: Edit craft formulas with 10 ingredients, success rates, outcomes, item-name lookup, and revision-backed saves
   - **Text File Editor**: Edit text-based configuration files
   - When editing spawn files, monster names are automatically displayed based on NPC ID
   - When viewing spawn files, map names are shown in brackets (e.g., "0.n_ndt (Wolfreck)")

8. **Monitor Metrics and Updates**: View system metrics on the dashboard with real-time charts (all authenticated users can view). If a newer stable Omnihance A3 Agent release is available, the dashboard shows a release banner with a GitHub link.
   - Use the top-right range selector above charts (default last hour) to switch between presets.
   - Chart density is automatically adjusted based on selected window to keep x-axis readable.

9. **File Revisions**: All file edits are automatically backed up. Use the revision system to revert changes if needed (requires admin or super admin role).

10. **Manage Server Processes** (Admin and Super Admin only):
    - Navigate to the Server Management page
    - Add processes by clicking "Add Process" or right-clicking executable/batch files in the file tree
    - Configure friendly names, paths, and optional ports
    - Reorder processes by clicking up/down arrows
    - Start/stop individual processes or the entire server sequence
    - Monitor real-time status and uptime for all processes
    - Viewers can access the page to see process status but cannot manage processes

11. **Manage SQL Server ODBC DSNs** (Admin and Super Admin only):
    - Navigate to the SQL ODBC page
    - Click "Create default ODBC values" to create the common A3 SQL Server User DSNs from a server name and user ID
    - Existing default DSNs are skipped so their current values are not overwritten
    - Add DSN name, SQL Server name, login ID, and default database
    - Test connection with a password before saving
    - Update or delete DSNs as needed

12. **Configure Game Server Settings** (Admin and Super Admin only):
    - Navigate to the Settings page
    - Configure game server database host, port, username, and password
    - Configure Zone, Account, Main, and Battle server directory paths using searchable suggestions
    - Server directory paths must contain `SvrInfo.ini` before they can be saved
    - Reuse saved database settings as local SQL Server backup defaults

13. **Use Server View**:
    - Navigate to Server View after configuring server paths
    - Click "Sync Data" as an Admin or Super Admin to read server files into the database
    - Review Main, Account, Zone, and Battle `SvrInfo.ini` cards
    - Open Main Map Info, Zone Loaded Maps, Monster Spawns, Monster Drops, and Shops to search synced server state
    - Upload client map, monster, and item data first when you want labels shown as `Name (ID)`

14. **Create and Run Backups** (Admin and Super Admin only):
    - Navigate to the Backups page
    - Create file/directory jobs or local SQL Server jobs
    - Leave the cron expression empty for manual-only backups, or add a cron expression for scheduled runs
    - Optionally set an archive password for encrypted ZIP output
    - Trigger jobs manually, cancel running jobs, and review paginated run history with output file downloads

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

- **Authentication & Authorization**:
  - Password hashing with bcrypt (cost factor: 10)
  - HTTP-only cookies for session management (prevents XSS attacks)
  - Signed cookies with secret key (prevents tampering)
  - Role-based access control (RBAC) with permission checks on all endpoints
  - Session validation on protected routes
  - User status validation (only active users can sign in)
- **Input Validation**:
  - Input validation on all endpoints using go-playground/validator
  - Email format validation
  - Password strength requirements (minimum 6 characters)
  - File path sanitization
- **Database Security**:
  - SQL injection prevention (parameterized queries with goqu)
  - Soft delete support for users (is_deleted flag)
- **Network Security**:
  - CORS configuration
  - Local IP checking middleware (optional)
- **Access Control**:
  - Permission-based endpoint protection
  - Super admin protection (cannot modify own status)
  - Status-based access restrictions

## License

See [LICENSE](LICENSE) file for details.

## Contributing

1. Write good readable and working code.
2. Write tests for new features.
3. Update documentation as needed.
4. Ensure all linting and formatting checks pass.

## Support

For issues, questions, or contributions, please refer to the project repository.
