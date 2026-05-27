// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  DefaultQueryExecutor,
  KyselyPlugin,
  CompiledQuery,
  QueryCompiler,
  DialectAdapter,
  ConnectionProvider,
  DatabaseConnection,
  QueryResult,
  QueryExecutor,
  RootOperationNode,
  UnknownRow,
  type AbortableOperationOptions,
  type AbortableQueryOptions,
} from 'kysely';
import type { QueryId } from 'kysely';

export class ChoysumQueryExecutor extends DefaultQueryExecutor implements QueryExecutor {
  #compiler: QueryCompiler;
  #adapter: DialectAdapter;
  #connectionProvider: ConnectionProvider;
  readonly #plugins: ReadonlyArray<KyselyPlugin>;

  constructor(compiler: QueryCompiler, adapter: DialectAdapter, connectionProvider: ConnectionProvider, plugins: KyselyPlugin[] = []) {
    super(compiler, adapter, connectionProvider, plugins);
    this.#compiler = compiler;
    this.#adapter = adapter;
    this.#connectionProvider = connectionProvider;
    this.#plugins = plugins;
  }

  get adapter(): DialectAdapter {
    return this.#adapter;
  }

  compileQuery(node: RootOperationNode, queryId: QueryId): CompiledQuery {
    return this.#compiler.compileQuery(node, queryId);
  }

  provideConnection<T>(consumer: (connection: DatabaseConnection) => Promise<T>, options?: AbortableOperationOptions): Promise<T> {
    return this.#connectionProvider.provideConnection(consumer, options);
  }

  withPlugins(plugins: ReadonlyArray<KyselyPlugin>): ChoysumQueryExecutor {
    return new ChoysumQueryExecutor(this.#compiler, this.#adapter, this.#connectionProvider, [...this.plugins, ...plugins]);
  }
  withPlugin(plugin: KyselyPlugin): ChoysumQueryExecutor {
    return new ChoysumQueryExecutor(this.#compiler, this.#adapter, this.#connectionProvider, [...this.plugins, plugin]);
  }
  withPluginAtFront(plugin: KyselyPlugin): ChoysumQueryExecutor {
    return new ChoysumQueryExecutor(this.#compiler, this.#adapter, this.#connectionProvider, [plugin, ...this.plugins]);
  }
  withConnectionProvider(connectionProvider: ConnectionProvider): ChoysumQueryExecutor {
    return new ChoysumQueryExecutor(this.#compiler, this.#adapter, connectionProvider, [...this.plugins]);
  }
  withoutPlugins(): ChoysumQueryExecutor {
    return new ChoysumQueryExecutor(this.#compiler, this.#adapter, this.#connectionProvider, []);
  }

  async executeQuery<R>(compiledQuery: CompiledQuery<unknown>, options?: AbortableQueryOptions): Promise<QueryResult<R>> {
    const qId = compiledQuery.queryId ?? ($choysum.xid.New() as unknown as QueryId);
    return this.provideConnection(async connection => {
      const result = await connection.executeQuery<R>(compiledQuery, options);
      return this.#transformResult(result, qId);
    }, options);
  }

  async #transformResult<T>(result: QueryResult<T>, queryId: QueryId): Promise<QueryResult<T>> {
    let transformed = result as QueryResult<UnknownRow>;
    for (const plugin of this.#plugins) {
      const transform = plugin.transformResult;
      if (transform) {
        transformed = await transform.call(plugin, { result: transformed, queryId });
      }
    }
    return transformed as QueryResult<T>;
  }
}
