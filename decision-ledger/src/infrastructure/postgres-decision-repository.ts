import type { Pool } from 'pg';
import type { Decision, DecisionStatus } from '../contracts.js';
import type { DecisionRepository } from '../application/decision-repository.js';

interface DecisionRow {
  id: string;
  title: string;
  status: DecisionStatus;
  context: string;
  decision: string;
  consequences: string;
  created_at: Date;
}

function toDecision(row: DecisionRow): Decision {
  return {
    id: row.id,
    title: row.title,
    status: row.status,
    context: row.context,
    decision: row.decision,
    consequences: row.consequences,
    createdAt: row.created_at.toISOString(),
  };
}

export class PostgresDecisionRepository implements DecisionRepository {
  constructor(private readonly pool: Pool) {}

  async list(status?: DecisionStatus): Promise<Decision[]> {
    const result =
      status === undefined
        ? await this.pool.query<DecisionRow>(
            `SELECT id, title, status, context, decision, consequences, created_at
             FROM decisions
             ORDER BY created_at, id`,
          )
        : await this.pool.query<DecisionRow>(
            `SELECT id, title, status, context, decision, consequences, created_at
             FROM decisions
             WHERE status = $1
             ORDER BY created_at, id`,
            [status],
          );

    return result.rows.map(toDecision);
  }

  async get(id: string): Promise<Decision | undefined> {
    const result = await this.pool.query<DecisionRow>(
      `SELECT id, title, status, context, decision, consequences, created_at
       FROM decisions
       WHERE id = $1`,
      [id],
    );
    const row = result.rows[0];
    return row ? toDecision(row) : undefined;
  }

  async save(decision: Decision): Promise<void> {
    await this.pool.query(
      `INSERT INTO decisions (id, title, status, context, decision, consequences, created_at)
       VALUES ($1, $2, $3, $4, $5, $6, $7)`,
      [
        decision.id,
        decision.title,
        decision.status,
        decision.context,
        decision.decision,
        decision.consequences,
        decision.createdAt,
      ],
    );
  }
}
