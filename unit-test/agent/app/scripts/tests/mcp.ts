/**
 * MCP Test Handlers
 * Simple handlers for testing MCP functionality
 */

/**
 * Ping - Simple ping tool
 * @param {Object} args - Arguments
 * @param {number} args.count - Number of pings (default: 1)
 * @param {string} args.message - Custom message (default: "ping")
 * @returns {Object} Ping response
 */
function Ping(args: any): any {
  const count = args?.count || 1;
  const message = args?.message || "ping";

  return {
    message: message === "ping" ? "pong" : message,
    count: count,
    timestamp: new Date().toISOString(),
  };
}

/**
 * Status - Get server status
 * @param {Object} args - Arguments
 * @param {boolean} args.verbose - Show detailed status (default: false)
 * @returns {Object} Status response
 */
function Status(args: any): any {
  const verbose = args?.verbose || false;
  const time = new Date().toISOString();

  const basicStatus = {
    status: "online",
    uptime: 3600,
    time: time,
  };

  if (verbose) {
    return {
      ...basicStatus,
      version: "1.0.0",
      memory: "128MB",
      platform: "linux",
      nodeVersion: "v18.0.0",
    };
  }

  return basicStatus;
}

/**
 * Echo - Echo back a message
 * @param {Object} args - Arguments
 * @param {string} args.message - Message to echo
 * @param {boolean} args.uppercase - Convert to uppercase (default: false)
 * @param {Object} ctx - Agent context (extra parameter for Process transport)
 * @returns {Object} Echo response
 */
function Echo(args: any, ctx?: any): any {
  if (!args?.message) {
    throw new Error("message is required");
  }

  const message = args.message;
  const uppercase = args.uppercase || false;

  const response: any = {
    echo: uppercase ? message.toUpperCase() : message,
    uppercase: uppercase,
    length: message.length,
    timestamp: new Date().toISOString(),
  };

  // Include context information if available
  if (ctx) {
    const authorized = ctx.authorized || ctx.Authorized;

    response.context = {
      has_context: true,
      chat_id: ctx.chat_id || ctx.ChatID || null,
      assistant_id: ctx.assistant_id || ctx.AssistantID || null,
      locale: ctx.locale || ctx.Locale || null,
      authorized: authorized
        ? {
            user_id: authorized.user_id || authorized.UserID || null,
            tenant_id: authorized.tenant_id || authorized.TenantID || null,
          }
        : null,
    };
  } else {
    response.context = {
      has_context: false,
    };
  }

  return response;
}

/**
 * Info - Get server information (Resource)
 * @returns {Object} Server information
 */
function Info(): any {
  return {
    name: "Echo Test Server",
    version: "1.0.0",
    description: "Simple MCP server for testing",
    capabilities: ["ping", "status", "echo"],
    uptime: 7200,
    startTime: "2024-11-27T10:00:00.000Z",
    time: new Date().toISOString(),
  };
}

/**
 * Health - Get health check status (Resource)
 * @param {string} check - Check type (optional)
 * @returns {Object} Health status
 */
function Health(check?: string): any {
  const time = new Date().toISOString();

  const basicHealth = {
    status: "healthy",
    uptime: 3600,
    time: time,
  };

  if (check === "all") {
    return {
      ...basicHealth,
      checks: {
        memory: "ok",
        uptime: "ok",
        cpu: "ok",
        disk: "ok",
      },
      details: {
        memoryUsage: {
          heapUsed: 95,
          heapTotal: 128,
          rss: 150,
        },
      },
    };
  }

  return basicHealth;
}
