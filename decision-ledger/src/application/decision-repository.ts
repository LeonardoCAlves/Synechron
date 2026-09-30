import type { Decision, DecisionStatus } from '../contracts.js';

export interface DecisionRepository {
  list(status?: DecisionStatus): Promise<Decision[]>;
  get(id: string): Promise<Decision | undefined>;
  save(decision: Decision): Promise<void>;
}
