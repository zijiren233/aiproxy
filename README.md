<div align="center">
  <h1>AI Proxy</h1>
  <p>Next-generation AI gateway with OpenAI-compatible protocol</p>
  
  [![Release](https://img.shields.io/github/release/labring/aiproxy)](https://github.com/labring/aiproxy/releases)
  [![License](https://img.shields.io/github/license/labring/aiproxy)](https://github.com/labring/aiproxy/blob/main/LICENSE)
  [![Go Version](https://img.shields.io/github/go-mod/go-version/labring/aiproxy?filename=core%2Fgo.mod)](https://github.com/labring/aiproxy/blob/main/core/go.mod)
  [![Build Status](https://img.shields.io/github/actions/workflow/status/labring/aiproxy/release.yml?branch=main)](https://github.com/labring/aiproxy/actions)
  
  [English](./README.md) | [简体中文](./README.zh.md)
</div>

---

## 🚀 Overview

AI Proxy is a powerful, production-ready AI gateway that provides intelligent request routing, comprehensive monitoring, and seamless multi-tenant management. Built with OpenAI-compatible protocols, it serves as the perfect middleware for AI applications requiring reliability, scalability, and advanced features.

## ✨ Key Features

### 🔄 **Intelligent Request Management**

- **Smart Retry Logic**: Intelligent retry strategies with automatic error recovery
- **Priority-based Channel Selection**: Route requests based on channel priority and error rates
- **Load Balancing**: Efficiently distribute traffic across multiple AI providers

### 📊 **Comprehensive Monitoring & Analytics**

- **Real-time Alerts**: Proactive notifications for balance warnings, error rates, and anomalies
- **Detailed Logging**: Complete request/response tracking with audit trails
- **Advanced Analytics**: Request volume, error statistics, RPM/TPM metrics, and cost analysis
- **Channel Performance**: Error rate analysis and performance monitoring

### 🏢 **Multi-tenant Architecture**

- **Organization Isolation**: Complete separation between different organizations
- **Flexible Access Control**: Token-based authentication with subnet restrictions
- **Resource Quotas**: RPM/TPM limits and usage quotas per group
- **Custom Pricing**: Per-group model pricing and billing configuration

### 🤖 **MCP (Model Context Protocol) Support**

- **Public MCP Servers**: Ready-to-use MCP integrations
- **Organization MCP Servers**: Private MCP servers for organizations
- **Embedded MCP**: Built-in MCP servers with configuration templates
- **OpenAPI to MCP**: Automatic conversion of OpenAPI specs to MCP tools

### 🔧 **Advanced Capabilities**

- **Multi-format Support**: Text, image, audio, and document processing
- **Model Mapping**: Flexible model aliasing and routing
- **Prompt Caching**: Intelligent caching with billing support
- **Think Mode**: Support for reasoning models with content splitting
- **Built-in Tokenizer**: No external tiktoken dependencies

## 🏗️ Architecture

```mermaid
graph TB
    Client[Client Applications] --> Gateway[AI Proxy Gateway]
    Gateway --> Auth[Authentication & Authorization]
    Gateway --> Router[Intelligent Router]
    Gateway --> Monitor[Monitoring & Analytics]
    
    Router --> Provider1[OpenAI]
    Router --> Provider2[Anthropic]
    Router --> Provider3[Azure OpenAI]
    Router --> ProviderN[Other Providers]
    
    Gateway --> MCP[MCP Servers]
    MCP --> PublicMCP[Public MCP]
    MCP --> GroupMCP[Organization MCP]
    MCP --> EmbedMCP[Embedded MCP]
    
    Monitor --> Alerts[Alert System]
    Monitor --> Analytics[Analytics Dashboard]
    Monitor --> Logs[Audit Logs]
```

## 🚀 Quick Start

### Docker (Recommended)

```bash
# Quick start with default configuration
docker run -d \
  --name aiproxy \
  -p 3000:3000 \
  -v $(pwd)/aiproxy:/aiproxy \
  ghcr.io/labring/aiproxy:latest

# Nightly build
docker run -d \
  --name aiproxy \
  -p 3000:3000 \
  -v $(pwd)/aiproxy:/aiproxy \
  ghcr.io/labring/aiproxy:main
```

### Docker Compose

```bash
# Download docker-compose.yaml
curl -O https://raw.githubusercontent.com/labring/aiproxy/main/docker-compose.yaml

# Start services
docker-compose up -d
```

## 🔧 Configuration

### Environment Variables

#### **Core Settings**

```bash
LISTEN=:3000                    # Server listen address
ADMIN_KEY=your-admin-key        # Admin API key
```

#### **Database Configuration**

```bash
SQL_DSN=postgres://user:pass@host:5432/db    # Primary database
LOG_SQL_DSN=postgres://user:pass@host:5432/log_db  # Log database (optional)
REDIS_CONN_STRING=redis://localhost:6379     # Redis for caching
```

#### **Feature Toggles**

```bash
BILLING_ENABLED=true           # Enable billing features
ENABLE_MODEL_ERROR_AUTO_BAN=true  # Auto-ban problematic models
SAVE_ALL_LOG_DETAIL=false     # Log all request details
```

### Advanced Configuration

<details>
<summary>Click to expand advanced configuration options</summary>

#### **Rate Limiting & Quotas**

```bash
GROUP_MAX_TOKEN_NUM=100        # Max tokens per group
MODEL_ERROR_AUTO_BAN_RATE=0.3  # Error rate threshold for auto-ban
```

#### **Logging & Retention**

```bash
LOG_STORAGE_HOURS=168          # Log retention (0 = unlimited)
LOG_DETAIL_STORAGE_HOURS=72    # Detail log retention
CLEAN_LOG_BATCH_SIZE=2000      # Log cleanup batch size
```

#### **Security & Access Control**

```bash
IP_GROUPS_THRESHOLD=5          # IP sharing alert threshold
IP_GROUPS_BAN_THRESHOLD=10     # IP sharing ban threshold
```

</details>

## 📚 API Documentation

### Interactive API Explorer

Visit `http://localhost:3000/swagger/index.html` for the complete API documentation with interactive examples.

### Quick API Examples

#### **List Available Models**

```bash
curl -H "Authorization: Bearer your-token" \
  http://localhost:3000/v1/models
```

#### **Chat Completion**

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## 🔌 Integrations

### Sealos Platform

Deploy instantly on Sealos with built-in model capabilities:
[Deploy to Sealos](https://hzh.sealos.run/?openapp=system-aiproxy)

### FastGPT Integration

Seamlessly integrate with FastGPT for enhanced AI workflows:
[FastGPT Documentation](https://doc.tryfastgpt.ai/docs/development/modelconfig/ai-proxy/)

### MCP (Model Context Protocol)

AI Proxy provides comprehensive MCP support for extending AI capabilities:

- **Public MCP Servers**: Community-maintained integrations
- **Organization MCP Servers**: Private organizational tools
- **Embedded MCP**: Easy-to-configure built-in functionality
- **OpenAPI to MCP**: Automatic tool generation from API specifications

## 🛠️ Development

### Prerequisites

- Go 1.24+
- Node.js 22+ (for frontend development)
- PostgreSQL/MySQL (optional, SQLite by default)
- Redis (optional, for caching)

### Building from Source

```bash
# Clone repository
git clone https://github.com/labring/aiproxy.git
cd aiproxy

# Build frontend (optional)
cd web && npm install -g pnpm && pnpm install && pnpm run build && cp -r dist ../core/public/dist/ && cd ..

# Build backend
cd core && go build -o aiproxy .

# Run
./aiproxy
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Ways to Contribute

- 🐛 Report bugs and issues
- 💡 Suggest new features
- 📝 Improve documentation
- 🔧 Submit pull requests
- ⭐ Star the repository

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- OpenAI for the API specification
- The open-source community for various integrations
- All contributors and users of AI Proxy