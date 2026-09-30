import type { Decision, DecisionStatus } from '../contracts.js';
import type { DecisionRepository } from '../application/decision-repository.js';

export class InMemoryDecisionRepository implements DecisionRepository {
  private readonly decisions = new Map<string, Decision>();

  async list(status?: DecisionStatus): Promise<Decision[]> {
    return [...this.decisions.values()]
      .filter((decision) => status === undefined || decision.status === status)
      .sort((left, right) => left.createdAt.localeCompare(right.createdAt));
  }

  async get(id: string): Promise<Decision | undefined> {
    return this.decisions.get(id);
  }

  async save(decision: Decision): Promise<void> {
    this.decisions.set(decision.id, decision);
  }
}
