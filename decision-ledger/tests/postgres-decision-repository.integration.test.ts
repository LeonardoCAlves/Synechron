import { spawn } from 'node:child_process';
import { Pool } from 'pg';
import { afterAll, beforeAll, beforeEach, describe, expect, it } from 'vitest';
import { DecisionService } from '../src/application/decision-service.js';
import { PostgresDecisionRepository } from '../src/infrastructure/postgres-decision-repository.js';

const testDatabaseUrl = process.env.TEST_DATABASE_URL;
if (!testDatabaseUrl) {
  throw new Error('TEST_DATABASE_URL is required to run PostgreSQL integration tests.');
}

let pool: Pool | undefined;

function getPool(): Pool {
  if (!pool) {
    throw new Error('The PostgreSQL test pool has not been initialized.');
  }
  return pool;
}

function createDecisionInAbruptProcess(): Promise<{
  id: string;
  title: string;
  status: string;
  context: string;
  decision: string;
  consequences: string;
  createdAt: string;
}> {
  const serviceUrl = new URL('../dist/application/decision-service.js', import.meta.url).href;
  const repositoryUrl = new URL(
    '../dist/infrastructure/postgres-decision-repository.js',
    import.meta.url,
  ).href;
  const poolUrl = new URL('../dist/infrastructure/postgres-pool.js', import.meta.url).href;
  const script = `
    import { DecisionService } from ${JSON.stringify(serviceUrl)};
    import { PostgresDecisionRepository } from ${JSON.stringify(repositoryUrl)};
    import { createPostgresPool } from ${JSON.stringify(poolUrl)};

    const pool = createPostgresPool(process.env.TEST_DATABASE_URL);
    const service = new DecisionService(new PostgresDecisionRepository(pool));
    const decision = await service.create(JSON.parse(process.env.DECISION_INPUT));
    process.stdout.write(JSON.stringify(decision), () => process.exit(0));
  `;

  return new Promise((resolve, reject) => {
    const child = spawn(process.execPath, ['--input-type=module', '--eval', script], {
      env: {
        ...process.env,
        TEST_DATABASE_URL: testDatabaseUrl,
        DECISION_INPUT: JSON.stringify(sampleInput),
      },
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    let stdout = '';
    let stderr = '';
    child.stdout.setEncoding('utf8');
    child.stderr.setEncoding('utf8');
    child.stdout.on('data', (chunk: string) => {
      stdout += chunk;
    });
    child.stderr.on('data', (chunk: string) => {
      stderr += chunk;
    });
    child.once('error', reject);
    child.once('close', (code) => {
      if (code !== 0) {
        reject(new Error(`Abrupt writer process failed (${code}): ${stderr}`));
        return;
      }
      try {
        resolve(JSON.parse(stdout));
      } catch (error) {
        reject(error);
      }
    });
  });
}

const sampleInput = {
  title: 'Integration test: adopt asynchronous messaging',
  context: 'The order workflow currently blocks on downstream services.',
  decision: 'Use a durable message broker to decouple order processing.',
  consequences: 'Operations must own retries, delivery, and message observability.',
};

describe('PostgresDecisionRepository', () => {
  beforeAll(() => {
    pool = new Pool({ connectionString: testDatabaseUrl });
  });

  beforeEach(async () => {
    await getPool().query('TRUNCATE TABLE decisions');
  });

  afterAll(async () => {
    await pool?.end();
  });

  it('persists records across repository instances', async () => {
    const service = new DecisionService(new PostgresDecisionRepository(getPool()));
    const created = await service.create(sampleInput);
    const restartedRepository = new PostgresDecisionRepository(getPool());

    await expect(restartedRepository.get(created.id)).resolves.toEqual(created);
  });

  it('keeps committed decisions after the writer process exits without graceful shutdown', async () => {
    const created = await createDecisionInAbruptProcess();
    const restartedRepository = new PostgresDecisionRepository(getPool());

    await expect(restartedRepository.get(created.id)).resolves.toEqual(created);
  });

  it('filters records by status using the database', async () => {
    const service = new DecisionService(new PostgresDecisionRepository(getPool()));
    await service.create(sampleInput);
    await service.create({ ...sampleInput, status: 'accepted' });

    const accepted = await service.list('accepted');

    expect(accepted).toHaveLength(1);
    expect(accepted[0]?.status).toBe('accepted');
  });

  it('persists concurrent decision creations without losing or duplicating records', async () => {
    const service = new DecisionService(new PostgresDecisionRepository(getPool()));

    const created = await Promise.all(
      Array.from({ length: 25 }, (_, index) =>
        service.create({ ...sampleInput, title: `Concurrent decision ${index}` }),
      ),
    );
    const stored = await service.list();

    expect(stored).toHaveLength(25);
    expect(new Set(created.map(({ id }) => id)).size).toBe(25);
    expect(new Set(stored.map(({ id }) => id)).size).toBe(25);
  });

  it('surfaces PostgreSQL client failures instead of returning a success-shaped result', async () => {
    const endedPool = new Pool({ connectionString: testDatabaseUrl });
    await endedPool.end();
    const repository = new PostgresDecisionRepository(endedPool);

    await expect(repository.list()).rejects.toThrow();
  });
});
