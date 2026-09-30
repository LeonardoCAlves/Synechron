import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import prettier from 'prettier';
import * as z from 'zod/v4';
import { decisionSchema, mcpToolContracts } from '../src/contracts.js';

const contractDirectory = resolve('contracts');
const checkOnly = process.argv.includes('--check');

const decisionContract = {
  $schema: 'https://json-schema.org/draft/2020-12/schema',
  $id: 'https://example.com/decision-ledger/contracts/decision.schema.json',
  ...z.toJSONSchema(decisionSchema),
};

const mcpToolsContract = {
  protocol: 'Model Context Protocol',
  server: 'decision-ledger',
  tools: Object.values(mcpToolContracts).map(({ name, title, description, inputSchema }) => ({
    name,
    title,
    description,
    inputSchema: z.toJSONSchema(inputSchema, { io: 'input' }),
  })),
};

const generatedFiles = new Map<string, string>(
  await Promise.all(
    (
      [
        ['decision.schema.json', decisionContract],
        ['mcp-tools.schema.json', mcpToolsContract],
      ] as const
    ).map(async ([fileName, contract]): Promise<[string, string]> => {
      const prettierOptions = await prettier.resolveConfig(resolve(contractDirectory, fileName));
      return [
        fileName,
        await prettier.format(`${JSON.stringify(contract, null, 2)}\n`, {
          ...prettierOptions,
          parser: 'json',
        }),
      ];
    }),
  ),
);

if (!checkOnly) {
  await mkdir(contractDirectory, { recursive: true });
}

for (const [fileName, expectedContent] of generatedFiles) {
  const filePath = resolve(contractDirectory, fileName);
  if (checkOnly) {
    let actualContent: string;
    try {
      actualContent = await readFile(filePath, 'utf8');
    } catch {
      throw new Error(
        `Missing generated contract: contracts/${fileName}. Run npm run contracts:generate.`,
      );
    }
    if (actualContent !== expectedContent) {
      throw new Error(
        `Stale generated contract: contracts/${fileName}. Run npm run contracts:generate.`,
      );
    }
  } else {
    await writeFile(filePath, expectedContent, 'utf8');
  }
}

if (!checkOnly) {
  console.error('Generated contracts in contracts/.');
} else {
  console.error('Generated contracts are up to date.');
}
