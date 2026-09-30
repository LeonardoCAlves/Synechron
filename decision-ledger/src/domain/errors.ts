export class DecisionNotFoundError extends Error {
  constructor(id: string) {
    super(`Decision '${id}' was not found.`);
    this.name = 'DecisionNotFoundError';
  }
}
