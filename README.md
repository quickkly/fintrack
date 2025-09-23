# FinTrack - Finance Tracking CLI Tool

A Go-based CLI tool for financial transaction tracking and management with advanced filtering capabilities.

## Features

- 🔗 **API Integration** - Fetch transaction data via REST API
- 🔍 **Advanced Filtering** - Comprehensive filtering and sorting options
- 📊 **Session Management** - Secure authentication and session handling
- ⚙️ **Configurable** - Flexible configuration management
- 🛡️ **Secure** - Safe session and credential management
- 📝 **HTTP Logging** - Complete request/response logging for debugging

## Installation

### From Source

```bash
# Clone or download the source code
cd fintrack-bot
make build
make install
```

### Using Go

```bash
# Build from source
go build -o fintrack .
```

## Quick Start

1. **Initialize configuration:**
   ```bash
   fintrack init
   ```

2. **Login to Bend:**
   ```bash
   fintrack bend login
   ```

3. **Check available accounts:**
   ```bash
   fintrack bend accounts
   ```

4. **Fetch recent transactions:**
   ```bash
   fintrack bend fetch --days 7
   ```

5. **Fetch transactions with advanced filtering:**
   ```bash
   fintrack bend transactions --from "2024-01-01" --to "2024-12-31" --category-id "food"
   ```

## Command Reference

### Global Commands

```bash
fintrack init                           # Setup config directories and files
fintrack config show                    # Show current configuration
fintrack config set <key> <value>       # Set configuration values
```

### Bend Operations

```bash
fintrack bend check                     # Check session status
fintrack bend login                     # Interactive token setup
fintrack bend accounts                  # List available accounts
fintrack bend transactions              # Fetch last 30 days, all accounts
fintrack bend transactions --days 7    # Fetch last 7 days
fintrack bend transactions --from 2024-01-01 --to 2024-01-31
fintrack bend transactions --account-id "acc123"
```

### Advanced Filtering

```bash
fintrack bend transactions --category-id "food"                    # Filter by category
fintrack bend transactions --sort-by "amount" --sort-order "ASC"   # Custom sorting
fintrack bend transactions --include-detailed                      # Include detailed summaries
fintrack bend transactions --log-http                              # Enable HTTP logging
```

## Configuration

### Default Locations

- Config: `~/.config/fintrack/config.yaml`
- Session: `~/.config/fintrack/session.json`

### Configuration Example

```yaml
bend:
  base_url: "https://bend.example.com"
  rate_limit: "1s"
  session_file: "~/.config/fintrack/session.json"
  timeout: "30s"
  device_type: "Web"
  device_location: "India"
```



## Workflow Examples

### Daily Transaction Fetch

```bash
# Quick daily update
fintrack bend transactions --days 1

# Review with advanced filtering
fintrack bend transactions --days 3 --category-id "food" --sort-by "amount"
```

### Monthly Reconciliation

```bash
# Fetch full month
fintrack bend fetch --from 2024-01-01 --to 2024-01-31


# Validate and commit
fintrack validate
git add . && git commit -m "January 2024 transactions"
```

### Setup New Installation

```bash
# Initialize
fintrack init

# Configure Bend
fintrack bend login



# Test setup
fintrack bend check
fintrack bend accounts
```

## Development

### Building

```bash
make build          # Build binary
make dev            # Fast development build
make install        # Install to system
make clean          # Clean build artifacts
```

### Testing

```bash
make test           # Run tests
make lint           # Run linter
make fmt            # Format code
```

### Project Structure

```
fintrack/
├── cmd/                    # CLI commands
│   ├── root.go            # Root command
│   ├── init.go            # Init command
│   ├── config.go          # Config management
│   └── blend/             # Bend commands
├── internal/              # Internal packages
│   ├── blend/             # Bend client
│   └── config/            # Configuration
├── configs/               # Default configurations
└── main.go                # Entry point
```

## Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [Brotli](https://github.com/andybalholm/brotli) - HTTP compression support

## Environment Variables

```bash
export FINTRACK_CONFIG="/path/to/config.yaml"     # Custom config path
```


## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make changes and add tests
4. Run tests: `make test`
5. Format code: `make fmt`
6. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 Documentation: Available in the project files
- 🐛 Issues: Report through your preferred platform
- 💬 Discussions: Community support available


---

**Note:** This tool is designed to work with Bend financial service. Ensure you have appropriate access before using.