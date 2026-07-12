// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { DialectName } from '../../orm/repository/repository_dialect';
import type { BaseQueryCondition } from '../../orm/repository/types';
import type { ModelMetadata } from '../../orm/metadata/model';
import { withComputeRunAsExecution } from './runas';
import { withBridgeFrame } from './bridge';
import { createEntityBackedModelInstance, resolveInstanceHandler } from './handler_runtime';
import { asObjectRecord } from '../../../utils/object';

function isPromiseLike<T = unknown>(value: unknown): value is Promise<T> {
  return !!value && typeof (value as { then?: unknown }).then === 'function';
}

type SearchRewriteResolved =
  | {
      kind: 'domain';
      domain: BaseQueryCondition;
    }
  | {
      kind: 'sql';
      sql: unknown;
    };

export function rewriteSearchCondition(
  meta: ModelMetadata,
  fieldName: string,
  op: unknown,
  value: unknown,
  dialect: DialectName,
  mode = 'query'
): SearchRewriteResolved | undefined {
  if (!fieldName || fieldName.includes('.')) return;

  const modelLabel = String(meta.fullModelName || meta.modelName || meta.className || 'Unknown');
  const fieldMeta = meta.fields.get(fieldName);
  if (!fieldMeta) return;

  const computeHandler = meta.computeHandlers?.get(fieldName);
  const sqlComputeHandler = meta.sqlComputeHandlers?.get(fieldName);
  const legacyCompute = fieldMeta.column?.compute;

  const isVirtual = Boolean(sqlComputeHandler) || computeHandler?.store === false || legacyCompute?.store === false || fieldMeta.related?.store === false;

  const explicitSearchHandler = meta.searchHandlers?.get(fieldName)?.method;
  const legacySearch = typeof legacyCompute?.search === 'string' ? legacyCompute.search.trim() : '';
  const handlerName = String(explicitSearchHandler || legacySearch || '').trim();
  const fromExplicitSearchDecorator = Boolean(explicitSearchHandler);

  if (!handlerName) {
    if (!isVirtual) return;

    if (legacyCompute?.store === false) {
      throw new Error(`Virtual compute field ${modelLabel}.${fieldName} does not declare compute.search and cannot participate in query conditions`);
    }

    throw new Error(`SEARCH_HANDLER_REQUIRED: virtual field ${modelLabel}.${fieldName} requires an explicit @Search handler`);
  }

  const runAs = computeHandler?.runAs === 'sudo' || legacyCompute?.runAs === 'sudo' ? 'sudo' : 'user';

  const executeWithBridge = () => {
    if (fromExplicitSearchDecorator) {
      const instanceMethod = resolveInstanceHandler(meta, fieldName, handlerName, '@Search');
      const modelInstance = createEntityBackedModelInstance(meta, {} as Record<string, unknown>);
      const searchCtx = {
        field(name: string) {
          return String(name || '').trim();
        },
        op() {
          return op;
        },
        value<T = unknown>() {
          return value as T;
        },
        and(clauses: BaseQueryCondition[]) {
          return { And: clauses } as BaseQueryCondition;
        },
        or(clauses: BaseQueryCondition[]) {
          return { Or: clauses } as BaseQueryCondition;
        },
        cmp(left: unknown, operator: unknown, right: unknown) {
          return [String(left || ''), operator, right] as BaseQueryCondition;
        },
        dialect,
      };

      return withBridgeFrame(modelInstance as object, 'search', searchCtx, () => instanceMethod.call(modelInstance));
    }

    const legacyHandler = resolveLegacySearchHandler(meta, handlerName);
    if (!legacyHandler) {
      throw new Error(`compute.search handler not found: ${modelLabel}.${fieldName} -> ${handlerName}`);
    }

    return legacyHandler({
      field: fieldName,
      op,
      value,
      dialect: String(dialect || 'postgres') as DialectName,
      runAs,
    });
  };

  const raw = withComputeRunAsExecution(meta, fieldName, runAs, 'search', executeWithBridge, mode);
  if (isPromiseLike(raw)) {
    throw new Error(
      `compute.search handler returned a Promise, but the current query compilation path only supports synchronous handlers: ${modelLabel}.${fieldName}`
    );
  }

  const result = normalizeSearchHandlerResult(raw);
  if (!result) {
    throw new Error(`compute.search handler returned an invalid value and must provide exactly one of domain or sql: ${modelLabel}.${fieldName}`);
  }

  return result;
}

function normalizeSearchHandlerResult(raw: unknown): SearchRewriteResolved | undefined {
  const record = asObjectRecord(raw);
  const hasDomain = record?.domain != null;
  const hasSql = record?.sql != null;

  if (hasDomain && !hasSql) {
    return {
      kind: 'domain',
      domain: record!.domain as BaseQueryCondition,
    };
  }

  if (hasSql && !hasDomain) {
    return {
      kind: 'sql',
      sql: record!.sql,
    };
  }

  if (hasDomain || hasSql) {
    return;
  }

  if (raw == null) {
    return;
  }

  return {
    kind: 'domain',
    domain: raw as BaseQueryCondition,
  };
}

function resolveLegacySearchHandler(meta: ModelMetadata, methodName: string): ((ctx: unknown) => unknown) | undefined {
  const key = String(methodName || '').trim();
  if (!key) return;

  const modelCtorRecord = meta.type as unknown as {
    computeSearchHandlers?: Record<string, (...args: unknown[]) => unknown>;
    __computeSearchHandlers__?: Record<string, (...args: unknown[]) => unknown>;
    [key: string]: unknown;
  };

  const registry = modelCtorRecord.computeSearchHandlers || modelCtorRecord.__computeSearchHandlers__;
  if (registry && typeof registry[key] === 'function') {
    return registry[key].bind(meta.type);
  }

  if (typeof modelCtorRecord[key] === 'function') {
    return (modelCtorRecord[key] as (ctx: unknown) => unknown).bind(meta.type);
  }

  if (typeof meta.type?.prototype?.[key] === 'function') {
    const fn = meta.type.prototype[key];
    return (ctx: unknown) => fn.call(Object.create(meta.type.prototype), ctx);
  }

  return undefined;
}
