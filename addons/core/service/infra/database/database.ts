// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  Kysely,
  KyselyConfig,
  KyselyProps,
  QueryCreatorProps,
  CompiledQuery,
  Compilable,
  isCompilable,
  QueryExecutor,
  QueryExecutorProvider,
  InsertResult,
  DeleteResult,
  UpdateResult,
  MergeResult,
  InsertQueryNode,
  DeleteQueryNode,
  UpdateQueryNode,
  NoResultErrorConstructor,
  isNoResultErrorConstructor,
  NoResultError,
  QueryNode,
  type QueryId,
} from 'kysely';

import { ChoysumConnectionProvider } from './driver/connection-provider';
import { ChoysumQueryExecutor } from './driver/executor';

declare module 'kysely' {
  interface Kysely<DB> {
    savepoint(name: string): Promise<string>;
    rollbackToSavepoint(name: string): Promise<void>;
    releaseSavepoint(name: string): Promise<void>;
    withSavepoint<T>(callback: () => Promise<T>, name?: string): Promise<T>;
  }
}

export type DrainOuterGeneric<T> = [T] extends [unknown] ? T : never;
export type SimplifySingleResult<O> = O extends InsertResult
  ? O
  : O extends DeleteResult
    ? O
    : O extends UpdateResult
      ? O
      : O extends MergeResult
        ? O
        : Simplify<O> | undefined;

export type SimplifyResult<O> = O extends InsertResult ? O : O extends DeleteResult ? O : O extends UpdateResult ? O : O extends MergeResult ? O : Simplify<O>;
export type Simplify<T> = DrainOuterGeneric<{ [K in keyof T]: T[K] } & {}>;

export class ChoysumDatabase<DB> extends Kysely<DB> implements QueryExecutorProvider {
  readonly #props: KyselyProps;
  constructor(args: KyselyConfig) {
    let superProps: QueryCreatorProps;
    let props: KyselyProps;

    const dialect = args.dialect;

    const driver = dialect.createDriver();
    const compiler = dialect.createQueryCompiler();
    const adapter = dialect.createAdapter();

    const connectionProvider = new ChoysumConnectionProvider(driver);
    const executor = new ChoysumQueryExecutor(compiler, adapter, connectionProvider, args.plugins ?? []);

    superProps = { executor };
    props = {
      config: args,
      executor,
      dialect,
      driver: driver,
    };
    super(args);
    this.#props = Object.freeze(props);
  }

  getExecutor(): QueryExecutor {
    return this.#props.executor;
  }

  /**
   * Execute a Compilable query and normalize its result.
   */
  async execute<R>(query: Compilable<R>): Promise<SimplifyResult<R>[]> {
    const compiledQuery = query.compile();
    const queryId = $choysum.xid.New() as unknown as QueryId;
    const result = await (this.getExecutor() as ChoysumQueryExecutor).executeQuery<R>(compiledQuery, queryId);
    const rows = result.rows as unknown as SimplifyResult<R>[];

    if (compiledQuery.query.kind === 'SelectQueryNode') {
      return rows;
    } else if (compiledQuery.query.kind === 'InsertQueryNode') {
      const queryNode = compiledQuery.query as InsertQueryNode;
      if ((queryNode.returning && this.getExecutor().adapter.supportsReturning) || (queryNode.output && this.getExecutor().adapter.supportsOutput)) {
        return rows;
      }
      return [new InsertResult(result.insertId, result.numAffectedRows ?? BigInt(0)) as unknown as SimplifyResult<R>];
    } else if (compiledQuery.query.kind === 'DeleteQueryNode') {
      const queryNode = compiledQuery.query as DeleteQueryNode;
      if ((queryNode.returning && this.getExecutor().adapter.supportsReturning) || (queryNode.output && this.getExecutor().adapter.supportsOutput)) {
        return rows;
      }
      return [new DeleteResult(result.numAffectedRows ?? BigInt(0)) as unknown as SimplifyResult<R>];
    } else if (compiledQuery.query.kind === 'UpdateQueryNode') {
      const queryNode = compiledQuery.query as UpdateQueryNode;
      if ((queryNode.returning && this.getExecutor().adapter.supportsReturning) || (queryNode.output && this.getExecutor().adapter.supportsOutput)) {
        return rows;
      }
      return [new UpdateResult(result.numAffectedRows ?? BigInt(0), result.numChangedRows) as unknown as SimplifyResult<R>];
    }
    return rows;
  }

  /**
   * Create a savepoint.
   * @param name Savepoint name.
   * @returns The savepoint name.
   */
  async savepoint(name: string): Promise<string> {
    const executor = this.getExecutor() as ChoysumQueryExecutor;
    return executor.provideConnection(async conn => {
      return await conn.savepoint(name);
    });
  }

  /**
   * Roll back to a savepoint.
   * @param name Savepoint name.
   */
  async rollbackToSavepoint(name: string): Promise<void> {
    const executor = this.getExecutor() as ChoysumQueryExecutor;
    return executor.provideConnection(async conn => {
      await conn.rollbackToSavepoint(name);
    });
  }

  /**
   * Release a savepoint on databases that support it.
   * @param name Savepoint name.
   */
  async releaseSavepoint(name: string): Promise<void> {
    const executor = this.getExecutor() as ChoysumQueryExecutor;
    return executor.provideConnection(async conn => {
      await conn.releaseSavepoint(name);
    });
  }

  /**
   * Execute code under savepoint protection.
   * @param callback Callback to execute.
   * @param name Optional savepoint name.
   * @returns The callback result.
   */
  async withSavepoint<T>(callback: () => Promise<T>, name?: string): Promise<T> {
    const executor = this.getExecutor() as ChoysumQueryExecutor;
    return executor.provideConnection(async conn => {
      // Use $choysum.xid.New() to generate a unique savepoint name.
      const spName = name || `sp_${$choysum.xid.New()}`;
      return await conn.withSavepoint(callback, spName);
    });
  }
}
