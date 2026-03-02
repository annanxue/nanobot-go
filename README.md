<div align="center">
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

## 🖥️ Local Models (Ollama)

Run nanobot with your own local models using Ollama.

**1. Start your Ollama server**

```bash
ollama serve
```

**2. Install a model**

```bash
ollama pull llama3
```

**3. Configure** (`~/.nanobotgo/config.json`)

```json
{
  "agents": {
    "defaults": {
      "provider": "ollama"
    }
  },
  "providers": {
    "ollama": {
      "model": "llama3",
      "apiKey": "dummy",
      "apiBase": "http://localhost:11434/v1"
    }
  }
}
```

**4. Chat**

```bash
nanobotgo agent -m "Hello from my local LLM!"
```

## 🤖 Multiple Agents

Configure multiple agents in `config.json` and use `@agent_name` in chat to switch between them.

**1. Configure** (`~/.nanobotgo/config.json`)

```json
{
  "agents": {
    "defaults": {
      "provider": "ollama",
      "workspace": "/path/to/workspace",
      "maxToolIterations": 100
    },
    "agents": [
      {
        "name": "default",
        "provider": "ollama",
        "model": "llama3"
      },
      {
        "name": "coder",
        "provider": "ollama",
        "model": "codellama"
      },
      {
        "name": "writer",
        "provider": "ollama",
        "model": "llama3"
      }
    ]
  },
  "providers": {
    "ollama": {
      "model": "llama3",
      "apiKey": "dummy",
      "apiBase": "http://localhost:11434/v1"
    }
  }
}
```

**2. Run gateway**

```bash
nanobotgo gateway
```

**3. Chat with specific agent**

In the WebUI chat:
- `@coder` - Use the coder agent
- `@writer` - Use the writer agent
- Plain message - Use the default agent

| Property | Description |
|----------|-------------|
| `name` | Agent name (used in @mention) |
| `provider` | Provider name from providers config |
| `model` | Model to use (overrides provider default) |
| `workspace` | Working directory for this agent |
| `maxToolIterations` | Max tool call iterations |
| `temperature` | LLM temperature setting |
| `maxTokens` | Max tokens for LLM response |


## 💬 Chat Apps

Talk to your nanobot through Feishu anywhere.

| Channel | Setup |
|---------|-------|
| **Feishu** | Medium (app credentials) |

<details>
<summary><b>Feishu (飞书)</b></summary>

Uses **WebSocket** long connection — no public IP required.

**1. Create a Feishu bot**
- Visit [Feishu Open Platform](https://open.feishu.cn/app)
- Create a new app → Enable **Bot** capability
- **Permissions**: Add `im:message` (send messages)
- **Events**: Add `im.message.receive_v1` (receive messages)
  - Select **Long Connection** mode (requires running nanobot first to establish connection)
- Get **App ID** and **App Secret** from "Credentials & Basic Info"
- Publish the app

**2. Configure**

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "appId": "cli_xxx",
      "appSecret": "xxx",
      "encryptKey": "",
      "verificationToken": "",
      "allowFrom": []
    }
  }
}
```

> `encryptKey` and `verificationToken` are optional for Long Connection mode.
> `allowFrom`: Leave empty to allow all users, or add `["ou_xxx"]` to restrict access.

**3. Run**

```bash
nanobotgo gateway
```

> [!TIP]
> Feishu uses WebSocket to receive messages — no webhook or public IP needed!

</details>

