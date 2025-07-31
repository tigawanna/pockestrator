# Pockestrator - PocketBase Instance Manager

## Project Overview

A Go application using PocketBase as a framework to manage multiple PocketBase instances on Linux systems. This project automates the deployment, configuration, and monitoring of PocketBase instances with systemd services and Caddy reverse proxy integration.

## Core Requirements

The application should replicate the functionality of the following bash script while providing a programmatic interface:

```sh
#!/usr/bin/env bash
#  for oracle ampere
project_name="moots"
port="8094"
version="0.28.4"
pocketbase_url="https://github.com/pocketbase/pocketbase/releases/download/v${version}/pocketbase_${version}_linux_amd64.zip"

echo "========= downloading pocketbase version ${version} ======="
wget -q "$pocketbase_url"
echo "========= unzipping pocketbase version ${version} ======="

sudo apt install zip -y
sudo mkdir -p /home/ubuntu/${project_name}

sudo unzip -q pocketbase_${version}_linux_amd64.zip -d /home/ubuntu/${project_name}

sudo chmod +x /home/ubuntu/${project_name}/pocketbase
echo "========= pocketbase version ${version} has been downloaded and unzipped into /home/ubuntu/${project_name} successfully! ======="

sudo rm -rf pocketbase_${version}_linux_amd64.zip

echo "========= setting up a systemd service ======= "
# setup a systemd service service
sudo touch /lib/systemd/system/${project_name}-pocketbase.service
echo "
[Unit]
Description = ${project_name} pocketbase

[Service]
Type           = simple
User           = root
Group          = root
LimitNOFILE    = 4096
Restart        = always
RestartSec     = 5s
StandardOutput   = append:/home/ubuntu/${project_name}/errors.log
StandardError    = append:/home/ubuntu/${project_name}/errors.log
WorkingDirectory = /home/ubuntu/${project_name}/
ExecStart      = /home/ubuntu/${project_name}/pocketbase serve --http="127.0.0.1:${port}"

[Install]
WantedBy = multi-user.target
" | sudo tee /lib/systemd/system/${project_name}-pocketbase.service



sudo systemctl daemon-reload
sudo systemctl enable ${project_name}-pocketbase.service
sudo systemctl start ${project_name}-pocketbase

echo "========= creating default superuser ======="
# Wait a moment for the service to fully start
sleep 3
# Create default superuser
cd /home/ubuntu/${project_name}
sudo ./pocketbase superuser upsert denniskinuthiaw@gmail.com denniskinuthiaw@gmail.com

echo "========= adding caddy configuration ======="
# Add subdomain configuration to Caddyfile
caddy_config="
${project_name}.tigawanna.vip {
    request_body {
        max_size 10MB
    }
    reverse_proxy 127.0.0.1:${port} {
        transport http {
            read_timeout 360s
        }
        # Add these headers to forward client IP
        header_up X-Forwarded-For {remote_host}
        header_up X-Real-IP {remote_host}
    }
}
"

# Check if Caddyfile exists and add configuration
if [ -f "/etc/caddy/Caddyfile" ]; then
    echo "Adding ${project_name} subdomain configuration to Caddyfile..."
    echo "$caddy_config" | sudo tee -a /etc/caddy/Caddyfile
    echo "Reloading Caddy configuration..."
    sudo systemctl reload caddy
else
    echo "Warning: /etc/caddy/Caddyfile not found. Please add the following configuration manually:"
    echo "$caddy_config"
fi

echo "========= setup complete! ======="
echo "Project: ${project_name}"
echo "Port: ${port}"
echo "Subdomain: ${project_name}.tigawanna.vip"
echo "Service: ${project_name}-pocketbase.service"
```


## Architecture Requirements

### Go Package Structure
Organize the project into well-defined Go packages:

1. **Service Management Package**
   - PocketBase instance creation and configuration
   - Version management and downloads
   - Binary extraction and permissions setup

2. **SystemD Integration Package**
   - Service file generation and management
   - Service lifecycle operations (enable/start/stop/restart)
   - Process monitoring and health checks

3. **Caddy Configuration Package**
   - Configuration file parsing and updates
   - Subdomain routing setup
   - SSL/TLS certificate management
   - Reverse proxy configuration

4. **Validation Package**
   - Configuration consistency checks
   - Port availability validation
   - System requirements verification

5. **Database Package**
   - PocketBase collection management
   - Service registry and tracking
   - Configuration history and audit logs

### Input Parameters & Validation

**Required Parameters:**
- `project_name`: Unique identifier for the service

**Optional Parameters:**
- `pocketbase_version`: Defaults to latest available version
- `port`: Defaults to 8091 or increments from last used port (8091 → 8092 → 8093...)
- `domain`: Base domain for subdomain generation (defaults to configured domain)

**Validation Rules:**
- Project names must be unique across all instances
- Ports must be unique and within valid range (1024-65535)
- Version numbers must exist in PocketBase releases
- Domain names must be valid and accessible

### PocketBase Framework Integration

Implement using PocketBase's Go framework with the following components:

#### Event Hooks Implementation
```go
// Service lifecycle hooks
app.OnRecordCreate("services").Add(func(e *core.RecordCreateEvent) error {
    // Create systemd service file
    // Update Caddy configuration
    // Deploy PocketBase instance
    return nil
})

app.OnRecordUpdate("services").Add(func(e *core.RecordUpdateEvent) error {
    // Validate configuration changes
    // Update system configurations
    return nil
})

app.OnRecordDelete("services").Add(func(e *core.RecordDeleteEvent) error {
    // Clean up systemd service
    // Remove Caddy configuration
    // Archive instance data
    return nil
})
```

#### Database Schema
**Services Collection:**
```json
{
  "name": "services",
  "schema": [
    {"name": "project_name", "type": "text", "required": true, "unique": true},
    {"name": "port", "type": "number", "required": true, "unique": true},
    {"name": "pocketbase_version", "type": "text", "required": true},
    {"name": "domain", "type": "text", "required": true},
    {"name": "status", "type": "select", "options": ["active", "inactive", "error"]},
    {"name": "systemd_config_hash", "type": "text"},
    {"name": "caddy_config_hash", "type": "text"},
    {"name": "last_health_check", "type": "date"},
    {"name": "created_by", "type": "relation", "relatedCollection": "users"}
  ]
}
```

#### Scheduled Jobs
```go
// Health check job
app.OnJobsScheduled().Add(func(jobsMap map[string]cron.Cron) error {
    jobsMap["health_check"] = func() {
        // Check service status
        // Validate configurations
        // Update database records
    }
    return nil
})
```

### System Integration Requirements

#### Prerequisite Management
- Verify Caddy installation and configuration
- Ensure systemd is available and functional
- Check user permissions for service management
- Validate network port availability

#### Service Deployment Process
1. **Download & Extract**: Fetch specified PocketBase version
2. **Directory Setup**: Create service directory structure
3. **Binary Setup**: Set proper permissions and ownership
4. **SystemD Service**: Generate and install service files
5. **Caddy Configuration**: Update reverse proxy settings
6. **Service Start**: Enable and start the service
7. **Superuser Creation**: Initialize admin user
8. **Health Verification**: Confirm service is running correctly

#### Configuration Management
- Generate systemd service files with proper security settings
- Create Caddy reverse proxy configurations with headers
- Implement configuration templates for consistency
- Support for custom configuration overrides

### Error Handling & Recovery

#### Validation Errors
- Duplicate project names or ports
- Invalid PocketBase versions
- Insufficient system resources
- Permission or access issues

#### Deployment Failures
- Download or extraction errors
- Service creation failures
- Configuration conflicts
- Network or firewall issues

#### Recovery Mechanisms
- Rollback failed deployments
- Clean up partial installations
- Restore previous configurations
- Detailed error logging and reporting

### Binary Distribution

The final application must be:
- Compiled as a single Go binary
- Include embedded web assets from `/dashboard/dist`
- Support cross-platform deployment (focus on Linux amd64/arm64)
- Include version information and build metadata
- Support CLI commands for headless operation

### Security Considerations

- Run services with minimal required privileges
- Implement proper file permissions and ownership
- Secure API endpoints with authentication
- Validate all user inputs and configurations
- Implement audit logging for all operations

### Performance Requirements

- Support concurrent service management operations
- Efficient configuration file parsing and updates
- Minimal resource overhead for monitoring
- Fast service deployment (under 30 seconds per instance)
- Scalable to manage 50+ PocketBase instances

This enhanced architecture provides a robust foundation for the PocketBase instance manager while maintaining clean separation of concerns and following Go best practices.



