# Namor 🚀

> A lightweight, webhook-based container orchestration tool for small teams and hobby projects

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-In%20Progress-yellow.svg)](#)

## 🎯 Why Namor?

Traditional container orchestration solutions like **Kubernetes** and **Watchtower** are:
- 🐘 **Bulky** - Overkill for small projects and staging environments
- ⏰ **Schedule-based** - Deploy on their own schedule, not when you need it
- 🔧 **Complex** - Time-consuming to configure for simple CI/CD workflows

**Namor** solves these problems by providing:
- ⚡ **Instant deployments** - Deploy immediately when images are created (like Vercel or Render)
- 🪶 **Lightweight** - Minimal resource footprint
- 🎣 **Webhook-driven** - Real-time responses to image updates
- 🔄 **Zero-downtime** - Smooth container transitions with minimal service interruption

Perfect for staging environments, hobby projects, and small teams who want modern CI/CD without the complexity.

## 🌟 Features

### Current Features (v1.0.0)
- 🐳 **Docker Runtime Support** - Full Docker integration with daemon monitoring
- 🎣 **Webhook Listener** - Real-time image update notifications
- 📦 **Container Orchestration** - Pull, deploy, and manage containers programmatically
- 🔄 **Minimal Downtime Deployment** - Smart container replacement strategies
- 📊 **Circuit Breaker Pattern** - Resilient container management
- 🏗️ **CLI Interface** - Easy-to-use command-line tool
- 📝 **Structured Logging** - Comprehensive logging with Zap
- ⚙️ **Configuration Management** - YAML-based service configuration

### Planned Features
- 🔮 **Containerd Support** - Extend beyond Docker
- 🟢 **Podman Integration** - Support for daemonless containers
- 🔍 **Service Discovery** - Automatic service registration and discovery  
- 📈 **Metrics & Monitoring** - Built-in observability
- 🌐 **Multi-Host Support** - Deploy across multiple nodes
- 🔐 **Advanced Security** - RBAC and security policies

## 🏗️ Architecture

```
┌───────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Image Registry  │    │     Webhook      │    │   Container     │
│   (GitHub/Docker  │────┤     Listener     ├────┤   Runtime       │
│   Hub/etc.)       │    │                  │    │   (Docker)      │
└───────────────────┘    └──────────────────┘    └─────────────────┘
                                 │
                                 ▼
                       ┌──────────────────┐
                       │   Orchestrator   │
                       │   - Pull Image   │
                       │   - Deploy       │
                       │   - Zero Downtime│
                       └──────────────────┘
```

## 🚀 Quick Start

### Prerequisites
- Go 1.24+ 
- Docker (running daemon)
- Git

### Installation

#### Option 1: Build from Source
```bash
# Clone the repository
git clone https://github.com/rndmcodeguy20/namor.git
cd namor

# Build the binary
go build -o namor main.go

# Make it executable (Linux/macOS)
chmod +x namor
```

#### Option 2: Download Release Binary
```bash
# Download from releases page (when available)
wget https://github.com/rndmcodeguy20/namor/releases/latest/download/namor-linux-amd64
chmod +x namor-linux-amd64
mv namor-linux-amd64 /usr/local/bin/namor
```

### Basic Usage

1. **Create a configuration file** (`namor.yml`):
```yaml
port: 5000
timeout: 10

webhook:
  url: https://your-webhook-endpoint.com/deploy
  secret: your_webhook_secret

runtime: docker
host: unix:///var/run/docker.sock  # Linux/macOS
# host: npipe:////./pipe/docker_engine  # Windows

services:
  my-app:
    requires_auth: true
    registry: ghcr.io/yourusername
    image: my-app
    name: my-app-container
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production
      - PORT=3000
    volumes:
      - ./app_data:/app/data
```

2. **Start Namor**:
```bash
./namor --config=namor.yml
```

3. **Set up webhooks** in your CI/CD pipeline or image registry to notify Namor when new images are available.

## 📖 Configuration

### Service Configuration Schema

```yaml
services:
  service-name:
    requires_auth: boolean          # Whether registry requires authentication
    registry: string               # Registry URL (e.g., ghcr.io/user)
    image: string                  # Image name
    user: string                   # User:Group for container (optional)
    name: string                   # Container name
    ports:                         # Port mappings
      - "host:container"
    environment:                   # Environment variables
      - "KEY=value"
    volumes:                       # Volume mounts
      - "host_path:container_path"
```

### CLI Flags

```bash
./namor --help

Flags:
  --config string     Configuration file path (default: namor.yml)
  --port int          Webhook listener port (default: 5000)
  --runtime string    Container runtime (docker, containerd, podman)
  --host string       Container runtime host connection
  --timeout int       Deployment timeout in seconds (default: 10)
  --verbose           Enable verbose logging
```

## 🔧 Development

### Project Structure
```
namor/
├── cmd/                    # CLI commands and entry points
│   ├── cli.go             # Main CLI logic
│   └── webhook.go         # Webhook handler
├── internal/              # Private application code
│   ├── config/           # Configuration management
│   ├── flags/            # CLI flag definitions
│   ├── helpers/          # Helper utilities
│   │   └── runtime/      # Container runtime abstractions
│   └── services/         # Business logic services
│       └── orchestrator/ # Container orchestration logic
├── pkg/                   # Public packages
│   ├── container_types/  # Container type definitions
│   ├── errors/           # Custom error types
│   └── utils/            # Utility functions
├── tests/                # Test configurations and fixtures
└── build/                # Build artifacts
```

### Building

```bash
# Build for current platform
go build -o build/namor main.go

# Build for multiple platforms
make build-all  # (if Makefile exists)

# Or manually:
GOOS=linux GOARCH=amd64 go build -o build/namor-linux-amd64 main.go
GOOS=windows GOARCH=amd64 go build -o build/namor-windows-amd64.exe main.go
GOOS=darwin GOARCH=amd64 go build -o build/namor-darwin-amd64 main.go
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Test with the example configuration
./namor --config=tests/namor.test.yml
```

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 🎯 Use Cases

### Perfect For:
- 🧪 **Staging Environments** - Quick deployments for testing
- 🏠 **Hobby Projects** - Simple container management without complexity
- 👥 **Small Teams** - Lightweight CI/CD for small development teams
- 🚀 **Rapid Prototyping** - Fast iteration cycles
- 📱 **Side Projects** - Learning container orchestration concepts

### When to Use Kubernetes Instead:
- Large-scale production deployments
- Complex multi-service architectures
- Enterprise-grade security requirements
- Advanced networking and service mesh needs
- High availability and disaster recovery requirements

## 🤝 Community & Support

This is a **fun side project** focused on implementing low-level container orchestration concepts from scratch. It's designed for learning and practical use in small-scale environments.

- 🐛 **Issues**: Report bugs and request features
- 💬 **Discussions**: Share ideas and ask questions
- 📖 **Wiki**: Detailed documentation and tutorials
- 🌟 **Star**: If you find this project useful!

## 📋 Roadmap

### Phase 1: Core Features (Current)
- [x] Docker runtime integration
- [x] Basic webhook listener
- [x] Container orchestration
- [x] CLI interface
- [ ] Complete webhook implementation
- [ ] Documentation improvements

### Phase 2: Extended Runtime Support
- [ ] Containerd integration
- [ ] Podman support
- [ ] Runtime selection and switching

### Phase 3: Advanced Features
- [ ] Service discovery
- [ ] Load balancing
- [ ] Health checks
- [ ] Metrics and monitoring
- [ ] Web UI dashboard

### Phase 4: Enterprise Features
- [ ] Multi-host deployment
- [ ] Security policies
- [ ] Backup and recovery
- [ ] API versioning

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by the complexity of Kubernetes and the simplicity needed for small projects
- Built with love for the developer community
- Special thanks to the Go ecosystem and container runtime maintainers

---

**Made with ❤️ for developers who want simple container orchestration**

*Namor: When Kubernetes is a spaceship and you just need a bicycle* 🚲