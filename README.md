<div align="center">
  <img src="nanobot_logo.png" alt="nanobot" width="500">
  <h1>nanobot-go: Ultra-Lightweight Personal AI Assistant With Go nanobotgo</h1>
</div>


## 📦 Install

**Install from source** (latest features, recommended for development)

```bash
go build -o nanobotgo .
```

## 🚀 Quick Start

**1. Initialize**

```bash
nanobotgo onboard
```

**2. Configure** (`config.json`)

For OpenRouter - recommended for global users:
```json
{
  "providers": {
    "openrouter": {
      "apiKey": "sk-or-v1-xxx"
    }
  },
  "agents": {
    "defaults": {
      "model": "anthropic/claude-opus-4-5"
    }
  }
}
```

**3. Chat**

```bash
nanobotgo agent -m "What is 2+2?"
```

That's it! You have a working AI assistant in 1 minutes.

