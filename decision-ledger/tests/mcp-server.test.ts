import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { InMemoryTransport, LATEST_PROTOCOL_VERSION } from '@modelcontextprotocol/server';
import type { McpServer } from '@modelcontextprotocol/server';
import { DecisionService } from '../src/application/decision-service.js';
import { InMemoryDecisionRepository } from '../src/infrastructure/in-memory-decision-repository.js';
import { createMcpServer } from '../src/mcp/server.js';

const validInput = {
  title: 'Adopt asynchronous messaging',
  context: 'The order workflow currently blocks on several downstream services.',
  decision: 'Use a durable message broker to decouple order processing from consumers.',
  consequences: 'Operations must own broker availability, retries, and message observability.',
};

type RpcResponse = {
  error?: { code: number; message: string };
  result?: Record<string, unknown>;
};

class McpTestClient {
  private readonly serverTransport: ReturnType<typeof InMemoryTransport.createLinkedPair>[number];
  private readonly transport: ReturnType<typeof InMemoryTransport.createLinkedPair>[number];
  private readonly pending = new Map<number, (message: RpcResponse) => void>();
  private nextId = 0;

  constructor() {
    [this.serverTransport, this.transport] = InMemoryTransport.createLinkedPair();
    this.transport.onmessage = (message) => {
      if ('id' in message && typeof message.id === 'number') {
        this.pending.get(message.id)?.(message as RpcResponse);
      }
    };
  }

  async connect(server: McpServer): Promise<void> {
    await server.connect(this.serverTransport);
    await this.transport.start();
  }

  async initialize(): Promise<void> {
    await this.request('initialize', {
      protocolVersion: LATEST_PROTOCOL_VERSION,
      capabilities: {},
      clientInfo: { name: 'decision-ledger-tests', version: '1.0.0' },
    });
    await this.transport.send({ jsonrpc: '2.0', method: 'notifications/initialized' });
  }

  async request(method: string, params?: Record<string, unknown>): Promise<RpcResponse> {
    const id = this.nextId++;
    const response = new Promise<RpcResponse>((resolve) => this.pending.set(id, resolve));
    await this.transport.send({
      jsonrpc: '2.0',
      id,
      method,
      ...(params === undefined ? {} : { params }),
    });
    const result = await response;
    this.pending.delete(id);
    return result;
  }

  async close(): Promise<void> {
    await this.transport.close();
  }
}

let server: McpServer;
let client: McpTestClient;

beforeEach(async () => {
  const service = new DecisionService(new InMemoryDecisionRepository());
  server = createMcpServer(service);
  client = new McpTestClient();
  await client.connect(server);
  await client.initialize();
});

afterEach(async () => {
  await server.close();
  await client.close();
});

describe('MCP server adapter', () => {
  it('discovers the declared tools, resources, resource templates, and prompt', async () => {
    const [tools, resources, templates, prompts] = await Promise.all([
      client.request('tools/list'),
      client.request('resources/list'),
      client.request('resources/templates/list'),
      client.request('prompts/list'),
    ]);

    expect((tools.result?.tools as { name: string }[]).map(({ name }) => name)).toEqual([
      'list-decisions',
      'get-decision',
      'create-decision',
    ]);
    expect((resources.result?.resources as { uri: string }[]).map(({ uri }) => uri)).toContain(
      'decisions://all',
    );
    expect(
      (templates.result?.resourceTemplates as { uriTemplate: string }[]).map(
        ({ uriTemplate }) => uriTemplate,
      ),
    ).toContain('decisions://{id}');
    expect((prompts.result?.prompts as { name: string }[]).map(({ name }) => name)).toContain(
      'review-decision',
    );
  });

  it('creates decisions and exposes them consistently through tools, resources, and prompts', async () => {
    const createdResponse = await client.request('tools/call', {
      name: 'create-decision',
      arguments: validInput,
    });
    const createdResult = createdResponse.result as {
      content: { text: string }[];
      isError?: boolean;
    };
    const created = JSON.parse(createdResult.content[0]!.text) as {
      id: string;
      title: string;
      status: string;
    };

    expect(createdResult.isError).toBeUndefined();
    expect(created).toMatchObject({ ...validInput, status: 'proposed' });

    const listed = await client.request('tools/call', {
      name: 'list-decisions',
      arguments: {},
    });
    expect(JSON.parse((listed.result as { content: { text: string }[] }).content[0]!.text)).toEqual(
      [created],
    );

    const resource = await client.request('resources/read', {
      uri: `decisions://${created.id}`,
    });
    expect((resource.result?.contents as { text: string }[])[0]?.text).toContain(created.id);

    const prompt = await client.request('prompts/get', {
      name: 'review-decision',
      arguments: { id: created.id },
    });
    expect((prompt.result?.messages as { content: { text: string } }[])[0]?.content.text).toContain(
      created.title,
    );
  });

  it('rejects invalid tool arguments at the protocol boundary', async () => {
    const response = await client.request('tools/call', {
      name: 'create-decision',
      arguments: { ...validInput, unexpected: 'not allowed' },
    });

    expect((response.result as { isError?: boolean }).isError).toBe(true);
  });

  it('reports missing decisions as tool errors and resource-not-found errors', async () => {
    const id = '00000000-0000-4000-8000-000000000000';
    const tool = await client.request('tools/call', {
      name: 'get-decision',
      arguments: { id },
    });
    expect((tool.result as { isError?: boolean }).isError).toBe(true);

    const resource = await client.request('resources/read', { uri: `decisions://${id}` });
    expect(resource.error?.code).toBe(-32602);
  });
});
