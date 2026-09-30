import { Pool } from 'pg';

export function createPostgresPool(databaseUrl = process.env.DATABASE_URL): Pool {
  if (!databaseUrl) {
    throw new Error('DATABASE_URL is required.');
  }

  const parsedUrl = new URL(databaseUrl);
  if (!['postgres:', 'postgresql:'].includes(parsedUrl.protocol)) {
    throw new Error('DATABASE_URL must use the PostgreSQL protocol.');
  }

  const pool = new Pool({
    connectionString: databaseUrl,
    max: 10,
    connectionTimeoutMillis: 5_000,
    idleTimeoutMillis: 30_000,
  });

  pool.on('error', (error: Error) => {
    console.error('Unexpected error from an idle PostgreSQL client.', error);
  });

  return pool;
}
