import { describe, expect, it } from 'vitest';
import { createDecisionSchema, decisionSchema } from '../src/contracts.js';

const validDecision = {
  id: '00000000-0000-4000-8000-000000000000',
  title: 'Adopt asynchronous messaging',
  status: 'accepted',
  context: 'The order workflow currently blocks on several downstream services.',
  decision: 'Use a durable message broker to decouple order processing from consumers.',
  consequences: 'Operations must own broker availability, retries, and message observability.',
  createdAt: '2026-01-15T10:30:00.000Z',
};

describe('decision contracts', () => {
  it('accepts a valid decision record', () => {
    expect(decisionSchema.safeParse(validDecision).success).toBe(true);
  });

  it('rejects records with unknown fields', () => {
    expect(decisionSchema.safeParse({ ...validDecision, internalNote: 'unexpected' }).success).toBe(
      false,
    );
  });

  it('rejects incomplete creation input', () => {
    expect(createDecisionSchema.safeParse({ title: 'Too short' }).success).toBe(false);
  });
});
