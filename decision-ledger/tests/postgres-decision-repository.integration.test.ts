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

  it('filters records by status using the database', async () => {
    const service = new DecisionService(new PostgresDecisionRepository(getPool()));
    await service.create(sampleInput);
    await service.create({ ...sampleInput, status: 'accepted' });

    const accepted = await service.list('accepted');

    expect(accepted).toHaveLength(1);
    expect(accepted[0]?.status).toBe('accepted');
  });
});
