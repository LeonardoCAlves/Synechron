import { DecisionService } from './application/decision-service.js';
import { PostgresDecisionRepository } from './infrastructure/postgres-decision-repository.js';
import { createPostgresPool } from './infrastructure/postgres-pool.js';
import { runMcpServer } from './mcp/server.js';

async function main(): Promise<void> {
  const pool = createPostgresPool();
  try {
    await pool.query('SELECT 1');
    const repository = new PostgresDecisionRepository(pool);
    const service = new DecisionService(repository);
    runMcpServer(service, () => pool.end());
  } catch (error) {
    await pool.end();
    throw error;
  }
}

main().catch((error: unknown) => {
  console.error('Decision Ledger MCP server failed to start.', error);
  process.exitCode = 1;
});
