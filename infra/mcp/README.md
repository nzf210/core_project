# WCH E2E MCP Server

Model Context Protocol server for browser automation via Playwright.
Enables Hermes (and any MCP-compatible client) to drive WCH Platform UI for end-to-end testing.

## Quick Start

```bash
cd infra/mcp
npm install
npm start              # launches MCP server on stdio
```

## Tools Exposed

| Tool | Purpose |
|:-----|:--------|
| `e2e_navigate` | Navigate browser to a URL |
| `e2e_click` | Click an element by CSS selector |
| `e2e_fill` | Fill a text input by CSS selector |
| `e2e_screenshot` | Save PNG screenshot to /tmp |
| `e2e_expect_selector` | Assert selector visibility/attachment |

All tools reuse a single Playwright page instance — call `e2e_navigate` first.

## AC-3 Example Flow

```js
// From an MCP client:
await callTool('e2e_navigate', { url: 'http://localhost:3201/login' });
await callTool('e2e_fill',    { selector: 'input[name="phone"]', value: '+628123456789' });
await callTool('e2e_fill',    { selector: 'input[name="password"]', value: 'secret123' });
await callTool('e2e_click',   { selector: 'button[type="submit"]' });
await callTool('e2e_navigate',{ url: 'http://localhost:3201/chatbot-config' });
await callTool('e2e_fill',    { selector: '#system-prompt', value: 'You are a helpful assistant.' });
await callTool('e2e_click',   { selector: 'button.save-config' });
await callTool('e2e_expect_selector', { selector: '.config-saved-toast', state: 'visible' });
await callTool('e2e_screenshot', {});
```

## Tests

```bash
npm test   # validates tool registry without launching browser
```

## Integration with Hermes

Add to `~/.hermes/mcp_servers.json`:

```json
{
  "mcpServers": {
    "wch-e2e": {
      "command": "node",
      "args": ["/path/to/wch-platform/infra/mcp/wch-e2e-server.js"]
    }
  }
}
```