import {
  McpServer,
  ProtocolError,
  ProtocolErrorCode,
  ResourceNotFoundError,
  ResourceTemplate,
} from '@modelcontextprotocol/server';
import { serveStdio } from '@modelcontextprotocol/server/stdio';
import * as z from 'zod/v4';
import { mcpToolContracts, reviewDecisionPromptSchema } from '../contracts.js';
import { DecisionNotFoundError } from '../domain/errors.js';
import type { DecisionService } from '../application/decision-service.js';

function textResult(value: unknown) {
  return { content: [{ type: 'text' as const, text: JSON.stringify(value, null, 2) }] };
}

function errorResult(error: unknown) {
  if (error instanceof DecisionNotFoundError) {
    return { content: [{ type: 'text' as const, text: error.message }], isError: true };
  }

  console.error('Decision Ledger MCP tool failed.', error);
  const message = 'The operation could not be completed. Check server logs for details.';
  return { content: [{ type: 'text' as const, text: message }], isError: true };
}

export function createMcpServer(service: DecisionService): McpServer {
  const server = new McpServer({ name: 'decision-ledger', version: '1.0.0' });

  server.registerTool(
    mcpToolContracts.listDecisions.name,
    {
      title: mcpToolContracts.listDecisions.title,
      description: mcpToolContracts.listDecisions.description,
      inputSchema: mcpToolContracts.listDecisions.inputSchema,
    },
    async ({ status }) => {
      try {
        return textResult(await service.list(status));
      } catch (error) {
        return errorResult(error);
      }
    },
  );

  server.registerTool(
    mcpToolContracts.getDecision.name,
    {
      title: mcpToolContracts.getDecision.title,
      description: mcpToolContracts.getDecision.description,
      inputSchema: mcpToolContracts.getDecision.inputSchema,
    },
    async ({ id }) => {
      try {
        return textResult(await service.get(id));
      } catch (error) {
        return errorResult(error);
      }
    },
  );

  server.registerTool(
    mcpToolContracts.createDecision.name,
    {
      title: mcpToolContracts.createDecision.title,
      description: mcpToolContracts.createDecision.description,
      inputSchema: mcpToolContracts.createDecision.inputSchema,
    },
    async (input) => {
      try {
        return textResult(await service.create(input));
      } catch (error) {
        return errorResult(error);
      }
    },
  );

  server.registerResource(
    'all-decisions',
    'decisions://all',
    {
      title: 'All architecture decisions',
      description: 'Read-only snapshot of every recorded architecture decision.',
      mimeType: 'application/json',
    },
    async (uri) => {
      try {
        return {
          contents: [
            {
              uri: uri.href,
              mimeType: 'application/json',
              text: JSON.stringify(await service.list(), null, 2),
            },
          ],
        };
      } catch (error) {
        console.error('Decision Ledger resource read failed.', error);
        throw new Error('Unable to read architecture decisions.');
      }
    },
  );

  server.registerResource(
    'decision-by-id',
    new ResourceTemplate('decisions://{id}', { list: undefined }),
    {
      title: 'Architecture decision',
      description: 'Read-only decision record selected by UUID.',
      mimeType: 'application/json',
    },
    async (uri, variables) => {
      const parsedId = z.uuid().safeParse(String(variables.id));
      if (!parsedId.success) {
        throw new ProtocolError(
          ProtocolErrorCode.InvalidParams,
          'Decision identifier must be a UUID.',
        );
      }

      try {
        return {
          contents: [
            {
              uri: uri.href,
              mimeType: 'application/json',
              text: JSON.stringify(await service.get(parsedId.data), null, 2),
            },
          ],
        };
      } catch (error) {
        if (error instanceof DecisionNotFoundError) {
          throw new ResourceNotFoundError(uri.href);
        }
        console.error('Decision Ledger resource read failed.', error);
        throw new Error('Unable to read the architecture decision.');
      }
    },
  );

  server.registerPrompt(
    'review-decision',
    {
      title: 'Review an architecture decision',
      description:
        'Evaluate a decision for trade-offs, risks, reversibility, and missing evidence.',
      argsSchema: reviewDecisionPromptSchema,
    },
    async ({ id }) => {
      try {
        const decision = await service.get(id);
        return {
          messages: [
            {
              role: 'user' as const,
              content: {
                type: 'text' as const,
                text: [
                  'Review this architecture decision. Identify material risks, alternatives, reversibility, and missing evidence. Separate facts from assumptions and propose the smallest useful follow-up.',
                  JSON.stringify(decision, null, 2),
                ].join('\n\n'),
              },
            },
          ],
        };
      } catch (error) {
        const message =
          error instanceof DecisionNotFoundError ? error.message : 'Could not load the decision.';
        if (!(error instanceof DecisionNotFoundError)) {
          console.error('Decision Ledger prompt failed.', error);
        }
        return {
          messages: [
            {
              role: 'user' as const,
              content: { type: 'text' as const, text: message },
            },
          ],
        };
      }
    },
  );

  return server;
}

export function runMcpServer(service: DecisionService, dispose: () => Promise<void>): void {
  let disposePromise: Promise<void> | undefined;
  const disposeOnce = (): Promise<void> => {
    disposePromise ??= dispose();
    return disposePromise;
  };

  const handle = serveStdio(() => {
    const server = createMcpServer(service);
    server.server.onclose = () => {
      void disposeOnce();
    };
    return server;
  });

  const shutdown = (): void => {
    void handle
      .close()
      .then(disposeOnce)
      .catch((error: unknown) => {
        console.error('Decision Ledger MCP server shutdown failed.', error);
        process.exitCode = 1;
      });
  };

  process.once('SIGINT', shutdown);
  process.once('SIGTERM', shutdown);
}
