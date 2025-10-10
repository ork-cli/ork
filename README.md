# Ork 🚀

> Orchestrate your microservices with simplicity and power

Ork is a modern development orchestration tool that eliminates the pain of microservices development. Run services from anywhere, manage dependencies intelligently, and switch seamlessly between local and pre-built images.

## Why Ork?

- **Run from anywhere** - No more navigating to specific directories
- **Smart dependency resolution** - Start one service, get all dependencies
- **Mix local and remote** - Use `--local` for active development, `--dev` for speed
- **Beautiful CLI** - Always know what's happening with clear, colorful output
- **Git-aware** - Automatically detects repo states and acts accordingly
- **Doctor built-in** - Comprehensive health checks with auto-fix capabilities

## Quick Start
```bash
# Install (coming soon)
curl -sSL https://ork.sh | sh

# Initialize your project
ork init

# Start services
ork up                    # Start all services
ork up frontend api       # Start specific services
ork up --local frontend --dev api postgres  # Mix and match

# Check health
ork doctor                # Run comprehensive health checks
ork doctor --fix         # Auto-fix common issues

# View status
ork ps                   # See what's running
ork logs api --follow    # Tail logs
```

