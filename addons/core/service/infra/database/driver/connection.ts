// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { CompiledQuery, DatabaseConnection, QueryResult } from 'kysely';
import { isResultSetQuery } from './is-result-set-query';
import { newDbError, wrapDbError, DbErrCode } from '../error';

declare module 'kysely' {
  export interface DatabaseConnection {
    savepoint(name: string): Promise<string>;
    rollbackToSavepoint(name: string): Promise<void>;
    releaseSavepoint(name: string): Promise<void>;
    withSavepoint<T>(callback: () => Promise<T>, name?: string): Promise<T>;
  }
}

export class ChoysumConnection implements DatabaseConnection {
  // Kysely transaction support is implemented via savepoints over the implicit
  // request-scoped transaction managed by Go.
  #txSavepoints: string[] = [];

  /*        Async methods            */
  public async beginTransaction() {
    try {
      const spName = `tx_${$choysum.xid.New()}`;
      await $choysum.db.savepoint(spName);
      this.#txSavepoints.push(spName);
    } catch (error) {
      throw wrapDbError(error, {
        code: DbErrCode.TRANSACTION_BEGIN_FAILED,
        message: 'Failed to begin transaction',
      });
    }
  }

  public async commitTransaction() {
    try {
      const spName = this.#txSavepoints.pop();
      if (spName === undefined) {
        throw newDbError({
          code: DbErrCode.TRANSACTION_NOT_STARTED,
          message: 'Transaction not started',
        });
      }

      await $choysum.db.releaseSavepoint(spName);
    } catch (error) {
      throw wrapDbError(error, {
        code: DbErrCode.TRANSACTION_COMMIT_FAILED,
        message: 'Failed to commit transaction',
      });
    }
  }

  public async rollbackTransaction() {
    try {
      const spName = this.#txSavepoints.pop();
      if (spName === undefined) {
        throw newDbError({
          code: DbErrCode.TRANSACTION_NOT_STARTED,
          message: 'Transaction not started',
        });
      }

      await $choysum.db.rollbackToSavepoint(spName);
    } catch (error) {
      throw wrapDbError(error, {
        code: DbErrCode.TRANSACTION_ROLLBACK_FAILED,
        message: 'Failed to rollback transaction',
      });
    }
  }

  async executeQuery<O>(compiledQuery: CompiledQuery): Promise<QueryResult<O>> {
    try {
      if (isResultSetQuery(compiledQuery)) {
        // if the query is a SELECT query, use $choysum.db.query
        try {
          const r = await $choysum.db.query(compiledQuery.sql, JSON.stringify(compiledQuery.parameters));

          let result: QueryResult<O> = {
            rows: JSON.parse(r),
          };
          return result;
        } catch (error) {
          throw wrapDbError(error, {
            code: DbErrCode.QUERY_FAILED,
            message: 'Failed to execute query',
          });
        }
      } else {
        // use $choysum.db.execute for all other queries
        try {
          const r = await $choysum.db.execute(compiledQuery.sql, JSON.stringify(compiledQuery.parameters));

          const jsonR = JSON.parse(r);
          let result: QueryResult<O> = {
            numChangedRows: jsonR.RowsAffected,
            numAffectedRows: jsonR.RowsAffected,
            insertId: jsonR.LastInsertId,
            rows: [],
          };
          return result;
        } catch (error) {
          throw wrapDbError(error, {
            code: DbErrCode.EXECUTION_FAILED,
            message: 'Failed to execute query',
          });
        }
      }
    } catch (error) {
      throw wrapDbError(error, {
        code: DbErrCode.EXECUTION_FAILED,
        message: 'Failed to execute query',
      });
    }
  }

  async *streamQuery<O>(_compiledQuery: CompiledQuery, _chunkSize: number): AsyncIterableIterator<QueryResult<O>> {
    throw newDbError({
      code: DbErrCode.STREAMING_NOT_SUPPORTED,
      message: 'Choysum API does not support streaming',
    });
  }

  /**
   * Create a savepoint.
   * @param name Savepoint name.
   * @returns The savepoint name.
   */
  public async savepoint(name: string): Promise<string> {
    try {
      return await $choysum.db.savepoint(name);
    } catch (error) {
      throw wrapDbError(error, {
        code: DbErrCode.SAVEPOINT_CREATE_FAILED,
        message: `Failed to create savepoint: ${name}`,
      });
    }
  }

  /**
   * Roll back to a savepoint.
   * @param name Savepoint name.
   */
  public async rollbackToSavepoint(name: string): Promise<void> {
    try {
      await $choysum.db.rollbackToSavepoint(name);
    } catch (error) {
      throw wrapDbError(error, {
        code: DbErrCode.SAVEPOINT_ROLLBACK_FAILED,
        message: `Failed to rollback to savepoint: ${name}`,
      });
    }
  }

  /**
   * Release a savepoint on databases that support it.
   * @param name Savepoint name.
   */
  public async releaseSavepoint(name: string): Promise<void> {
    try {
      await $choysum.db.releaseSavepoint(name);
    } catch (error) {
      throw wrapDbError(error, {
        code: DbErrCode.SAVEPOINT_RELEASE_FAILED,
        message: `Failed to release savepoint: ${name}`,
      });
    }
  }

  /**
   * Execute an operation under savepoint protection and roll back automatically on failure.
   * @param callback Callback to execute.
   * @param name Optional savepoint name.
   * @returns The callback result.
   */
  public async withSavepoint<T>(callback: () => Promise<T>, name?: string): Promise<T> {
    // Use $choysum.xid.New() to generate a unique savepoint name.
    const spName = name || `sp_${$choysum.xid.New()}`;

    try {
      await this.savepoint(spName);
    } catch (error) {
      throw wrapDbError(error, {
        code: DbErrCode.SAVEPOINT_CREATE_FAILED,
        message: `Failed to create savepoint for withSavepoint: ${spName}`,
      });
    }

    try {
      const result = await callback();

      try {
        await this.releaseSavepoint(spName);
      } catch (error) {
        // Log release failures as warnings without affecting the main operation result.
        console.warn(`Warning: Failed to release savepoint ${spName}:`, error);
      }

      return result;
    } catch (error) {
      try {
        await this.rollbackToSavepoint(spName);
      } catch (rollbackError) {
        // If rollback also fails, wrap the original error and include rollback failure details.
        throw wrapDbError(error, {
          code: DbErrCode.SAVEPOINT_OPERATION_FAILED,
          message: `Operation failed and rollback to savepoint ${spName} also failed: ${rollbackError}`,
        });
      }

      // Re-throw the original error; wrapDbError decides whether additional wrapping is needed.
      throw wrapDbError(error, {
        code: DbErrCode.SAVEPOINT_OPERATION_FAILED,
        message: `Operation failed in savepoint ${spName}`,
      });
    }
  }
}
