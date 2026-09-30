import { describe, expect, it } from 'vitest';
import { DecisionService } from '../src/application/decision-service.js';
import { DecisionNotFoundError } from '../src/domain/errors.js';
import { InMemoryDecisionRepository } from '../src/infrastructure/in-memory-decision-repository.js';

const validInput = {
  title: 'Adopt asynchronous messaging',
  context: 'The order workflow currently blocks on several downstream services.',
  decision: 'Use a durable message broker to decouple order processing from consumers.',
  consequences: 'Operations must own broker availability, retries, and message observability.',
};

describe('DecisionService', () => {
  it('creates decisions with a proposed status and stable metadata', async () => {
    const service = new DecisionService(new InMemoryDecisionRepository());

    const created = await service.create(validInput);

    expect(created).toMatchObject({ ...validInput, status: 'proposed' });
    expect(created.id).toMatch(/^[0-9a-f-]{36}$/i);
    expect(Number.isNaN(Date.parse(created.createdAt))).toBe(false);
  });

  it('filters decisions by lifecycle status', async () => {
    const service = new DecisionService(new InMemoryDecisionRepository());
    await service.create(validInput);
    await service.create({ ...validInput, status: 'accepted' });

    const accepted = await service.list('accepted');

    expect(accepted).toHaveLength(1);
    expect(accepted[0]?.status).toBe('accepted');
  });

  it('returns a typed not-found error for unknown identifiers', async () => {
    const service = new DecisionService(new InMemoryDecisionRepository());

    await expect(service.get('00000000-0000-4000-8000-000000000000')).rejects.toBeInstanceOf(
      DecisionNotFoundError,
    );
  });
});
