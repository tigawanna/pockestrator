# Pockestrator 🚀

A robust Go binary for managing PocketBase instances on Linux with automated deployment, systemd service creation, Caddy reverse proxy configuration, and a comprehensive REST API for service management.

## ✨ Features

- **🔧 Full Service Orchestration**: Create, deploy, and manage PocketBase services
- **🖥️ Embedded React Dashboard**: Modern UI embedded in the Go binary
- **🔒 SystemD Integration**: Automatic service file creation and management
- **🌐 Caddy Integration**: Automatic reverse proxy configuration
- **📊 Real-time Monitoring**: Service status, logs, and health checks
- **✅ Comprehensive Validation**: System requirements and service configuration validation
- **🛡️ Permission Management**: Proper sudo integration for system operations
- **📝 Complete API Documentation**: Ready for UI integration

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   React UI      │────│  Go Binary       │────│  PocketBase     │
│  (Embedded)     │    │  (Orchestrator)  │    │   Instance      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
   ┌────▼────┐         ┌─────▼─────┐         ┌─────▼─────┐
   │ SystemD │         │   Caddy   │         │   File    │
   │Manager  │         │ Manager   │         │  System   │
   └─────────┘         └───────────┘         └───────────┘
```

## 🚀 Quick Start

### Prerequisites

1. **Go 1.21+** installed
2. **SystemD** available
3. **Caddy** installed (optional, for reverse proxy)
4. **Sudo permissions** for system operations

### Required Sudo Permissions

Create `/etc/sudoers.d/pockestrator`:
```bash
# Allow pockestrator user to manage systemd services
pockestrator ALL=(ALL) NOPASSWD: /bin/systemctl, /usr/bin/systemctl
pockestrator ALL=(ALL) NOPASSWD: /bin/journalctl, /usr/bin/journalctl
pockestrator ALL=(ALL) NOPASSWD: /usr/bin/systemd-analyze
```

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd pockestrator

# Build the dashboard
cd dashboard
pnpm install
pnpm build
cd ..

# Build the Go binary
go build

# Run Pockestrator
./pockestrator serve --http="127.0.0.1:8091"
```

### Access

- **Dashboard**: http://localhost:8091
- **API**: http://localhost:8091/api/pockestrator
- **Admin Panel**: http://localhost:8091/_/

## 📚 API Documentation

See [API_DOCUMENTATION.md](./API_DOCUMENTATION.md) for comprehensive endpoint documentation.

### Key Endpoints

- `GET /api/pockestrator/system/health` - System health check
- `GET /api/pockestrator/system/info` - System information
- `POST /api/pockestrator/services` - Create new service
- `GET /api/pockestrator/services` - List all services
- `GET /api/pockestrator/services/{id}/status` - Get service status
- `GET /api/pockestrator/services/{id}/logs` - Get service logs
- `POST /api/pockestrator/services/{id}/control` - Control service (start/stop/restart)

## 🛠️ Development

### Project Structure

```
pockestrator/
├── main.go                     # Main application entry point
├── dashboard/                  # React dashboard (embedded)
│   ├── src/                   # Dashboard source code
│   └── dist/                  # Built dashboard (embedded)
├── internal/                   # Internal packages
│   ├── service/               # Service management
│   ├── systemd/               # SystemD integration
│   ├── caddy/                 # Caddy configuration
│   ├── validation/            # System validation
│   └── database/              # Database operations
├── pkg/                       # Public packages
│   └── orchestrator.go        # Main orchestration logic
└── test/                      # Tests
```

### Key Components

1. **Service Manager** (`internal/service/`): Handles PocketBase service lifecycle
2. **SystemD Manager** (`internal/systemd/`): Manages systemd service files and operations
3. **Caddy Manager** (`internal/caddy/`): Manages Caddy reverse proxy configuration
4. **Validator** (`internal/validation/`): Validates system requirements and configurations
5. **Database Manager** (`internal/database/`): Handles database operations and records
6. **Orchestrator** (`pkg/orchestrator.go`): Coordinates all operations

### Building Dashboard

```bash
cd dashboard
pnpm install
pnpm build  # Creates dist/ folder that gets embedded
```

### Running Tests

```bash
go test ./...
```

## 🔧 Configuration

### Default Configuration

```go
BaseDir:       "/home/ubuntu"           # Base directory for services
SystemdDir:    "/lib/systemd/system"    # SystemD service files location
CaddyConfig:   "/etc/caddy/Caddyfile"   # Caddy configuration file
DefaultDomain: "tigawanna.vip"          # Default domain for services
```

### Environment Variables

- `POCKESTRATOR_BASE_DIR`: Override base directory
- `POCKESTRATOR_SYSTEMD_DIR`: Override systemd directory
- `POCKESTRATOR_CADDY_CONFIG`: Override Caddy config path
- `POCKESTRATOR_DEFAULT_DOMAIN`: Override default domain

## 🔍 Troubleshooting

### Permission Issues

If you encounter permission errors:

1. Ensure proper sudo configuration
2. Check file/directory permissions
3. Run system health check: `GET /api/pockestrator/system/health`

### Service Creation Failures

1. Check if port is available
2. Verify project name is unique
3. Ensure systemd directory is writable
4. Check PocketBase binary download

### Caddy Configuration Issues

1. Verify Caddy is installed
2. Check Caddyfile permissions
3. Validate Caddy syntax

## 📝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Update documentation
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License.

## 🙏 Acknowledgments

- [PocketBase](https://pocketbase.io/) - The amazing backend-as-a-service
- [Caddy](https://caddyserver.com/) - The ultimate server with automatic HTTPS
- [SystemD](https://systemd.io/) - The system and service manager

---

**Built with ❤️ for the PocketBase community**
