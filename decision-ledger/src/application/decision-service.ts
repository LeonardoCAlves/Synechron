import { randomUUID } from 'node:crypto';
import type { CreateDecisionInput, Decision, DecisionStatus } from '../contracts.js';
import { DecisionNotFoundError } from '../domain/errors.js';
import type { DecisionRepository } from './decision-repository.js';

export class DecisionService {
  constructor(private readonly repository: DecisionRepository) {}

  async list(status?: DecisionStatus): Promise<Decision[]> {
    return this.repository.list(status);
  }

  async get(id: string): Promise<Decision> {
    const decision = await this.repository.get(id);
    if (!decision) {
      throw new DecisionNotFoundError(id);
    }
    return decision;
  }

  async create(input: CreateDecisionInput): Promise<Decision> {
    const decision: Decision = {
      id: randomUUID(),
      title: input.title,
      status: input.status ?? 'proposed',
      context: input.context,
      decision: input.decision,
      consequences: input.consequences,
      createdAt: new Date().toISOString(),
    };

    await this.repository.save(decision);
    return decision;
  }
}
