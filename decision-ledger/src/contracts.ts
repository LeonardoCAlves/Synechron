import * as z from 'zod/v4';

export const decisionStatusSchema = z.enum(['proposed', 'accepted', 'rejected', 'superseded']);

export const decisionSchema = z.strictObject({
  id: z.uuid().describe('Stable unique decision identifier.'),
  title: z.string().min(8).max(120).describe('Short, imperative architecture decision title.'),
  status: decisionStatusSchema.describe('Lifecycle state of the decision.'),
  context: z
    .string()
    .min(20)
    .max(4000)
    .describe('Forces and conditions that motivate this decision.'),
  decision: z.string().min(20).max(4000).describe('The chosen option and its rationale.'),
  consequences: z
    .string()
    .min(20)
    .max(4000)
    .describe('Positive, negative, and follow-up consequences.'),
  createdAt: z.iso.datetime().describe('UTC creation timestamp in ISO 8601 format.'),
});

export const createDecisionSchema = z.strictObject({
  title: decisionSchema.shape.title,
  context: decisionSchema.shape.context,
  decision: decisionSchema.shape.decision,
  consequences: decisionSchema.shape.consequences,
  status: decisionStatusSchema
    .optional()
    .describe('Initial lifecycle state. Defaults to proposed when omitted.'),
});

export const listDecisionsInputSchema = z.strictObject({
  status: decisionStatusSchema.optional().describe('Optional lifecycle state filter.'),
});

export const getDecisionInputSchema = z.strictObject({
  id: z.uuid().describe('Stable unique decision identifier.'),
});

export const reviewDecisionPromptSchema = z.strictObject({
  id: z.uuid().describe('Identifier of the decision to review.'),
});

export const mcpToolContracts = {
  listDecisions: {
    name: 'list-decisions',
    title: 'List architecture decisions',
    description: 'List recorded architecture decisions, optionally filtered by lifecycle status.',
    inputSchema: listDecisionsInputSchema,
  },
  getDecision: {
    name: 'get-decision',
    title: 'Get an architecture decision',
    description: 'Retrieve a decision record by its UUID.',
    inputSchema: getDecisionInputSchema,
  },
  createDecision: {
    name: 'create-decision',
    title: 'Create an architecture decision',
    description: 'Record a proposed or finalized architecture decision.',
    inputSchema: createDecisionSchema,
  },
} as const;

export type Decision = z.infer<typeof decisionSchema>;
export type CreateDecisionInput = z.infer<typeof createDecisionSchema>;
export type DecisionStatus = z.infer<typeof decisionStatusSchema>;
