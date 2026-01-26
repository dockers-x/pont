# MCP Integration for Pont

[English](#english) | [中文](#中文) | [日本語](#日本語)

---

## English

### What is MCP?

MCP (Model Context Protocol) is a protocol that allows AI models to interact with external tools and services. Pont now supports MCP, enabling AI assistants to manage your tunnels programmatically.

### Features

Pont exposes two MCP tools:

1. **listTunnels** - List all available tunnel configurations with their current status
2. **startTunnel** - Start a specific tunnel by ID and get the public URL for external access

### MCP Endpoint

The MCP endpoint is available at:
```
http://localhost:13333/mcp
```

### Configuration

#### For Claude Desktop

Add the following to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "pont": {
      "url": "http://localhost:13333/mcp"
    }
  }
}
```

#### For Other MCP Clients

Configure your MCP client to connect to the SSE endpoint:
```
http://localhost:13333/mcp
```

### Usage Example

Once configured, you can ask your AI assistant:

- "List all available tunnels"
- "Start the tunnel with ID xxx-xxx-xxx"
- "Show me the public URL for my local server"

The AI will use the MCP tools to interact with Pont and manage your tunnels.

### Security Considerations

- The MCP endpoint is accessible on your local network
- Ensure proper network security measures are in place
- Consider using firewall rules to restrict access if needed

---

## 中文

### 什么是 MCP？

MCP（模型上下文协议）是一种允许 AI 模型与外部工具和服务交互的协议。Pont 现在支持 MCP，使 AI 助手能够以编程方式管理您的隧道。

### 功能

Pont 提供两个 MCP 工具：

1. **listTunnels** - 列出所有可用的隧道配置及其当前状态
2. **startTunnel** - 通过 ID 启动特定隧道并获取外部访问的公网 URL

### MCP 端点

MCP 端点地址：
```
http://localhost:13333/mcp
```

### 配置

#### Claude Desktop 配置

将以下内容添加到 Claude Desktop 配置文件：

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "pont": {
      "url": "http://localhost:13333/mcp"
    }
  }
}
```

#### 其他 MCP 客户端

配置您的 MCP 客户端连接到 SSE 端点：
```
http://localhost:13333/mcp
```

### 使用示例

配置完成后，您可以向 AI 助手询问：

- "列出所有可用的隧道"
- "启动 ID 为 xxx-xxx-xxx 的隧道"
- "显示我本地服务器的公网 URL"

AI 将使用 MCP 工具与 Pont 交互并管理您的隧道。

### 安全注意事项

- MCP 端点可在您的本地网络上访问
- 请确保采取适当的网络安全措施
- 如有需要，考虑使用防火墙规则限制访问

---

## 日本語

### MCP とは？

MCP（モデルコンテキストプロトコル）は、AI モデルが外部ツールやサービスと対話できるようにするプロトコルです。Pont は MCP をサポートしており、AI アシスタントがプログラムでトンネルを管理できるようになりました。

### 機能

Pont は 2 つの MCP ツールを提供します：

1. **listTunnels** - すべての利用可能なトンネル設定とその現在のステータスをリスト
2. **startTunnel** - ID で特定のトンネルを開始し、外部アクセス用のパブリック URL を取得

### MCP エンドポイント

MCP エンドポイントは以下で利用可能です：
```
http://localhost:13333/mcp
```

### 設定

#### Claude Desktop の場合

Claude Desktop 設定ファイルに以下を追加します：

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "pont": {
      "url": "http://localhost:13333/mcp"
    }
  }
}
```

#### その他の MCP クライアント

MCP クライアントを SSE エンドポイントに接続するように設定します：
```
http://localhost:13333/mcp
```

### 使用例

設定後、AI アシスタントに次のように尋ねることができます：

- "利用可能なすべてのトンネルをリストして"
- "ID xxx-xxx-xxx のトンネルを開始して"
- "ローカルサーバーのパブリック URL を表示して"

AI は MCP ツールを使用して Pont と対話し、トンネルを管理します。

### セキュリティに関する考慮事項

- MCP エンドポイントはローカルネットワークでアクセス可能です
- 適切なネットワークセキュリティ対策を講じてください
- 必要に応じて、ファイアウォールルールを使用してアクセスを制限することを検討してください
