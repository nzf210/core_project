// WCH Platform E2E MCP Server
// Model Context Protocol server for browser automation via Playwright
// Tools: navigate, click, fill, screenshot, expect_selector

const { Server } = require('@modelcontextprotocol/sdk/server/index.js');
const { StdioServerTransport } = require('@modelcontextprotocol/sdk/server/stdio.js');
const { CallToolRequestSchema, ListToolsRequestSchema } = require('@modelcontextprotocol/sdk/types.js');
const { chromium } = require('playwright');

const server = new Server(
  {
    name: 'wch-e2e-mcp',
    version: '1.0.0',
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

let browser = null;
let page = null;

async function getBrowser() {
  if (!browser) {
    browser = await chromium.launch({ headless: true });
  }
  return browser;
}

server.setRequestHandler(ListToolsRequestSchema, async () => {
  return {
    tools: [
      {
        name: 'e2e_navigate',
        description: 'Navigate to a URL in the browser',
        inputSchema: {
          type: 'object',
          properties: {
            url: { type: 'string', description: 'URL to navigate to' }
          },
          required: ['url']
        }
      },
      {
        name: 'e2e_click',
        description: 'Click an element by selector',
        inputSchema: {
          type: 'object',
          properties: {
            selector: { type: 'string', description: 'CSS selector to click' }
          },
          required: ['selector']
        }
      },
      {
        name: 'e2e_fill',
        description: 'Fill a text input by selector',
        inputSchema: {
          type: 'object',
          properties: {
            selector: { type: 'string', description: 'CSS selector for input element' },
            value: { type: 'string', description: 'Value to fill in' }
          },
          required: ['selector', 'value']
        }
      },
      {
        name: 'e2e_screenshot',
        description: 'Take screenshot of current page or element',
        inputSchema: {
          type: 'object',
          properties: {
            selector: { type: 'string', description: 'Optional CSS selector' }
          }
        }
      },
      {
        name: 'e2e_expect_selector',
        description: 'Assert that selector exists on page',
        inputSchema: {
          type: 'object',
          properties: {
            selector: { type: 'string', description: 'CSS selector to check' },
            state: { type: 'string', enum: ['visible', 'hidden', 'attached'], default: 'visible' }
          },
          required: ['selector']
        }
      }
    ]
  };
});

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;

  try {
    switch (name) {
      case 'e2e_navigate': {
        if (!page) {
          const browser = await getBrowser();
          page = await browser.newPage();
        }
        await page.goto(args.url);
        const title = await page.title();
        return {
          content: [{ type: 'text', text: `Navigated to ${args.url}. Title: ${title}` }]
        };
      }

      case 'e2e_click': {
        if (!page) throw new Error('No page loaded. Call e2e_navigate first.');
        await page.click(args.selector);
        return {
          content: [{ type: 'text', text: `Clicked selector: ${args.selector}` }]
        };
      }

      case 'e2e_fill': {
        if (!page) throw new Error('No page loaded. Call e2e_navigate first.');
        await page.fill(args.selector, args.value);
        return {
          content: [{ type: 'text', text: `Filled '${args.value}' into ${args.selector}` }]
        };
      }

      case 'e2e_screenshot': {
        if (!page) throw new Error('No page loaded. Call e2e_navigate first.');
        const timestamp = Date.now();
        let path;
        if (args.selector) {
          path = `/tmp/e2e-screenshot-${timestamp}.png`;
          await page.screenshot({ path, clip: await page.evaluate((sel) => {
            const el = document.querySelector(sel);
            const rect = el.getBoundingClientRect();
            return { x: rect.x, y: rect.y, width: rect.width, height: rect.height };
          }, args.selector) });
        } else {
          path = `/tmp/e2e-screenshot-${timestamp}.png`;
          await page.screenshot({ path });
        }
        return {
          content: [{ type: 'text', text: `Screenshot saved: ${path}` }]
        };
      }

      case 'e2e_expect_selector': {
        if (!page) throw new Error('No page loaded. Call e2e_navigate first.');
        const state = args.state || 'visible';
        try {
          await page.waitForSelector(args.selector, { state });
          return {
            content: [{ type: 'text', text: `Selector '${args.selector}' is ${state}` }]
          };
        } catch (e) {
          return {
            content: [{ type: 'text', text: `FAILED: Selector '${args.selector}' not ${state} - ${e.message}` }],
            isError: true
          };
        }
      }

      default:
        return {
          content: [{ type: 'text', text: `Unknown tool: ${name}` }],
          isError: true
        };
    }
  } catch (error) {
    return {
      content: [{ type: 'text', text: `Error: ${error.message}` }],
      isError: true
    };
  }
});

const transport = new StdioServerTransport();
server.connect(transport);