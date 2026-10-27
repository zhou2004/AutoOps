// Cordis ToolDefinition interface from the official Harness SDK. No shell or local files are exposed.
import { randomUUID } from 'node:crypto';
export const name = 'autoops-authorized-tools';
export const inject = ['tools'];
const names = ['docker_info', 'docker_list', 'docker_logs', 'docker_start', 'docker_stop', 'docker_restart'];
export function apply(ctx) {
  ctx.tools.guard(exec => names.includes(exec.name) ? undefined : 'Only authorized AutoOps tools are available');
  let calls = 0;
  for (const name of names) {
    ctx.tools.register({
      name,
      description: `${name}: operate on a selected AutoOps Docker asset. Use a complete container NAME for logs/start/stop/restart. The server enforces user permissions and time-limited session grants. Never retry an uncertain mutation.`,
      parameters: {
        type: 'object', additionalProperties: false,
        properties: { connectionId: { type: 'integer', minimum: 1 }, container: { type: 'string', maxLength: 128 } },
        required: ['connectionId'],
      },
      output: { schema: { type: 'string' }, render: (_args, value) => [{ type: 'text', text: value }] },
      timeoutMs: 90000,
      async execute(args, exec) {
        if (++calls > 20) throw new Error('Tool budget exhausted for this turn');
        if (!Number.isInteger(args?.connectionId) || args.connectionId < 1 || (args.container !== undefined && typeof args.container !== 'string')) throw new Error('Invalid tool arguments');
        const response = await fetch(process.env.AUTOOPS_CALLBACK_URL, {
          method: 'POST', redirect: 'error', signal: exec.signal,
          headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${process.env.AUTOOPS_CALLBACK_TOKEN}` },
          body: JSON.stringify({ callId: randomUUID(), tool: name, connectionId: args.connectionId, container: args.container || '' }),
        });
        const body = await response.text();
        if (body.length > 200000) throw new Error('Tool response exceeds limit');
        // A denied action remains a tool result the assistant can explain to the user.
        return body;
      },
    });
  }
}
