// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { DatabaseConnection } from 'kysely';
import { ChoysumQueryExecutor } from './executor';

function withPatchedChoysum<T>(fn: () => Promise<T>): Promise<T> {
  const key = '$choysum';
  const hadOwn = Object.prototype.hasOwnProperty.call(globalThis as object, key);
  const previous = (globalThis as Record<string, unknown>)[key];
  let seq = 0;

  (globalThis as Record<string, unknown>)[key] = {
    ...(previous as Record<string, unknown>),
    xid: {
      New: () => {
        seq += 1;
        return `qid_${seq}`;
      },
    },
  };

  return fn().finally(() => {
    if (hadOwn) {
      (globalThis as Record<string, unknown>)[key] = previous;
    } else {
      delete (globalThis as Record<string, unknown>)[key];
    }
  });
}

function createHarness(plugins: any[] = []) {
  const calls = {
    compile: [] as Array<{ node: any; queryId: any }>,
    provideConnection: 0,
    executeQuery: [] as any[],
  };

  const compiler = {
    compileQuery(node: any, queryId: any) {
      calls.compile.push({ node, queryId });
      return {
        sql: 'select 1',
        parameters: [],
        query: {
          kind: 'SelectQueryNode',
          node,
          queryId,
        },
      };
    },
  };

  const adapter = {
    supportsReturning: true,
    supportsOutput: true,
  };

  const connection: DatabaseConnection = {
    executeQuery: async (compiledQuery: any) => {
      calls.executeQuery.push(compiledQuery);
      return {
        rows: ['base'],
      } as any;
    },
    savepoint: async (name: string) => name,
    rollbackToSavepoint: async () => {},
    releaseSavepoint: async () => {},
    withSavepoint: async <T>(callback: () => Promise<T>) => await callback(),
    streamQuery: async function* () {
      yield {
        rows: [],
      } as any;
    },
  };

  const connectionProvider = {
    async provideConnection<T>(consumer: (connection: DatabaseConnection) => Promise<T>): Promise<T> {
      calls.provideConnection += 1;
      return await consumer(connection);
    },
  };

  const executor = new ChoysumQueryExecutor(compiler as any, adapter as any, connectionProvider as any, plugins as any);
  return { executor, compiler, adapter, connectionProvider, calls };
}

test('query executor exposes adapter and delegates compile/provideConnection behavior', async () => {
  const harness = createHarness();

  expect(harness.executor.adapter).toBe(harness.adapter as any);

  const compiled = harness.executor.compileQuery({ kind: 'node' } as any, 'qid_compile' as any);
  expect(compiled.sql).toBe('select 1');
  expect(harness.calls.compile.length).toBe(1);
  expect(harness.calls.compile[0]).toEqual({
    node: { kind: 'node' },
    queryId: 'qid_compile',
  });

  const provided = await harness.executor.provideConnection(async () => 'ok');
  expect(provided).toBe('ok');
  expect(harness.calls.provideConnection).toBe(1);
});

test('query executor plugin combinators append/prepend/replace plugins as expected', async () => {
  const calls: string[] = [];

  const pluginA = {
    transformQuery(args: any) {
      return args.node;
    },
    async transformResult({ result }: any) {
      calls.push('A');
      return { ...result, rows: [...(result.rows || []), 'A'] };
    },
  };
  const pluginB = {
    transformQuery(args: any) {
      return args.node;
    },
    async transformResult({ result }: any) {
      calls.push('B');
      return { ...result, rows: [...(result.rows || []), 'B'] };
    },
  };
  const pluginC = {
    transformQuery(args: any) {
      return args.node;
    },
    async transformResult({ result }: any) {
      calls.push('C');
      return { ...result, rows: [...(result.rows || []), 'C'] };
    },
  };
  const pluginNoop = {
    transformQuery(args: any) {
      return args.node;
    },
  };

  await withPatchedChoysum(async () => {
    const base = createHarness([pluginA, pluginNoop]).executor;

    calls.length = 0;
    const appended = base.withPlugin(pluginB);
    const appendedResult = await appended.executeQuery({ query: { kind: 'SelectQueryNode' } } as any, 'qid_manual' as any);
    expect(appendedResult.rows).toEqual(['base', 'A', 'B']);
    expect(calls).toEqual(['A', 'B']);

    calls.length = 0;
    const front = base.withPluginAtFront(pluginC);
    const frontResult = await front.executeQuery({ query: { kind: 'SelectQueryNode' } } as any, 'qid_front' as any);
    expect(frontResult.rows).toEqual(['base', 'C', 'A']);
    expect(calls).toEqual(['C', 'A']);

    calls.length = 0;
    const multi = base.withPlugins([pluginB, pluginC]);
    const multiResult = await multi.executeQuery({ query: { kind: 'SelectQueryNode' } } as any, 'qid_multi' as any);
    expect(multiResult.rows).toEqual(['base', 'A', 'B', 'C']);
    expect(calls).toEqual(['A', 'B', 'C']);

    const clean = base.withoutPlugins();
    const cleanResult = await clean.executeQuery({ query: { kind: 'SelectQueryNode' } } as any, 'qid_clean' as any);
    expect(cleanResult.rows).toEqual(['base']);
  });
});

test('query executor executeQuery uses manual queryId first and xid fallback when queryId is omitted', async () => {
  await withPatchedChoysum(async () => {
    const base = createHarness();

    const directProviderCalls: number[] = [];
    const altProvider = {
      async provideConnection<T>(consumer: (connection: DatabaseConnection) => Promise<T>): Promise<T> {
        directProviderCalls.push(1);
        return await consumer({
          executeQuery: async () => ({ rows: ['alt'] }) as any,
          savepoint: async (name: string) => name,
          rollbackToSavepoint: async () => {},
          releaseSavepoint: async () => {},
          withSavepoint: async <T>(callback: () => Promise<T>) => await callback(),
          streamQuery: async function* () {
            yield { rows: [] } as any;
          },
        } as unknown as DatabaseConnection);
      },
    };

    const withOtherProvider = base.executor.withConnectionProvider(altProvider as any);
    const manual = await withOtherProvider.executeQuery({ query: { kind: 'SelectQueryNode' } } as any, 'qid_explicit' as any);
    expect(manual.rows).toEqual(['alt']);
    expect(directProviderCalls.length).toBe(1);

    const auto = await base.executor.executeQuery({ query: { kind: 'SelectQueryNode' } } as any);
    expect(auto.rows).toEqual(['base']);
  });
});

test('query executor constructor defaults to empty plugins when plugins argument is omitted', async () => {
  const harness = createHarness([
    {
      transformQuery(args: any) {
        return args.node;
      },
      async transformResult() {
        throw new Error('plugin list should be empty when constructor plugins are omitted');
      },
    },
  ]);

  const executor = new ChoysumQueryExecutor(harness.compiler as any, harness.adapter as any, harness.connectionProvider as any);
  const result = await executor.executeQuery({ query: { kind: 'SelectQueryNode' } } as any, 'qid_default_plugins' as any);

  expect(result.rows).toEqual(['base']);
});
