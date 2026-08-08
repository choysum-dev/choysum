// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Conventional Model static service wrappers (depth / transport / deny-read).
 *
 * Author contract for draft `this` vs class-level RPC:
 * `modules/core/service/runtime/RECORD_LIFECYCLE_THIS.md`
 *
 * Full design:
 * `.dev/docs/core/service/record_lifecycle_proxy_wrapper_boundary_plan20260715.md`
 */
import { ServiceMetadata, ParamMetadata, MetadataStorage, ValueType } from '../metadata';
import BaseModel from '../model/model';
import { buildRelationAliasCandidates } from '../relation/relation_alias';
import { RepositoryFactory } from '../repository/repository_factory';
import { isDecimal, isDecimalLeak } from '../../../utils/decimal';
import { asObjectRecord, hasOwnKey } from '../../../utils/object';
import type { ObjectRecord } from '../../../utils/types';
import Decimal from '@/core/utils/decimal';
import { getProxyKind } from '../../runtime/proxy/brand';

const WRAPPED_CONVENTIONAL_SERVICE_METHODS = Symbol.for('choysum.wrappedConventionalServiceMethods');
const GENERATED_MODEL_SERVICE_DEFINITIONS = new Map<string, Map<string, ServiceMetadata>>();
const LOADED_MODEL_CTORS = new Map<string, unknown>();

const KEY_CANDIDATES_CACHE = new Map<string, string[]>();

export type GeneratedModelServiceDefinition = {
  name: string;
  kind?: ServiceMetadata['kind'];
  type?: ValueType;
  description?: string;
  params?: ParamMetadata[];
};

function cloneParams(params: ParamMetadata[] | undefined): ParamMetadata[] | undefined {
  if (!Array.isArray(params)) return undefined;
  return params.map(param => ({ ...param }));
}

function cloneServiceMetadata(input: ServiceMetadata): ServiceMetadata {
  const out: ServiceMetadata = {
    name: input.name,
    kind: input.kind,
    type: input.type,
  };

  if (input.description !== undefined) {
    out.description = input.description;
  }

  const params = cloneParams(input.params);
  if (params) {
    out.params = params;
  }

  return out;
}

function normalizeGeneratedServiceDefinition(input: GeneratedModelServiceDefinition): ServiceMetadata | undefined {
  const name = String(input?.name || '').trim();
  if (!name) return undefined;

  const normalized: ServiceMetadata = {
    name,
    kind: input.kind === 'ServerStream' ? 'ServerStream' : 'Unary',
    type: input.type || 'Struct',
  };

  if (input.description !== undefined) {
    normalized.description = input.description;
  }

  const params = cloneParams(input.params);
  if (params) {
    normalized.params = params;
  }

  return normalized;
}

function applyGeneratedServiceDefinitionsToModel(fullModelName: string, modelCtor: unknown): void {
  const definitions = GENERATED_MODEL_SERVICE_DEFINITIONS.get(fullModelName);
  if (!definitions || definitions.size === 0 || !modelCtor || typeof modelCtor !== 'function') return;

  const meta = MetadataStorage.instance.getModelMetadata(modelCtor as Parameters<typeof MetadataStorage.instance.getModelMetadata>[0]);
  const nextServices = new Map(meta.services);

  definitions.forEach((serviceMeta, methodName) => {
    nextServices.set(methodName, cloneServiceMetadata(serviceMeta));
  });

  const patch = { services: nextServices } as unknown as Parameters<typeof MetadataStorage.instance.setModelMetadata>[1];
  MetadataStorage.instance.setModelMetadata(modelCtor as Parameters<typeof MetadataStorage.instance.setModelMetadata>[0], patch);
}

export function registerGeneratedModelServiceDefinitions(fullModelName: string, definitions: GeneratedModelServiceDefinition[]): void {
  const key = String(fullModelName || '').trim();
  if (!key || !Array.isArray(definitions)) return;

  let bucket = GENERATED_MODEL_SERVICE_DEFINITIONS.get(key);
  if (!bucket) {
    bucket = new Map<string, ServiceMetadata>();
    GENERATED_MODEL_SERVICE_DEFINITIONS.set(key, bucket);
  }

  for (const definition of definitions) {
    const normalized = normalizeGeneratedServiceDefinition(definition);
    if (!normalized) continue;
    bucket.set(normalized.name, normalized);
  }

  const loadedModelCtor = LOADED_MODEL_CTORS.get(key);
  if (loadedModelCtor) {
    applyGeneratedServiceDefinitionsToModel(key, loadedModelCtor);
  }
}

export function registerLoadedModelForGeneratedServiceMetadata(fullModelName: string, modelCtor: unknown): void {
  const key = String(fullModelName || '').trim();
  if (!key || !modelCtor || typeof modelCtor !== 'function') return;

  LOADED_MODEL_CTORS.set(key, modelCtor);
  applyGeneratedServiceDefinitionsToModel(key, modelCtor);
}

function toSnakeCase(s: string): string {
  return s
    .replace(/([A-Z])/g, '_$1')
    .replace(/^_+/, '')
    .toLowerCase();
}

function buildKeyCandidates(key: string): string[] {
  const cached = KEY_CANDIDATES_CACHE.get(key);
  if (cached) return cached;

  const lc = key ? key.charAt(0).toLowerCase() + key.slice(1) : key;
  const snake = toSnakeCase(key);
  const list = Array.from(new Set([key, lc, snake].filter(Boolean)));
  KEY_CANDIDATES_CACHE.set(key, list);
  return list;
}

type ServiceState = { depth: number; [k: string]: unknown };

function parseMaybeJsonObject(v: unknown): unknown {
  if (typeof v !== 'string') return v;
  const c = v.charCodeAt(0);
  const looksJson = c === 123 /* { */ || c === 91; /* [ */
  if (!looksJson) return v;
  try {
    return JSON.parse(v);
  } catch {
    return v;
  }
}

function getCurrentChoysumRequest(): unknown {
  const globalRecord = asObjectRecord(globalThis);
  const choysum = asObjectRecord(globalRecord?.$choysum);
  return choysum?.request;
}

function getOrInitServiceState(req: unknown): ServiceState {
  const reqRecord = asObjectRecord(req);
  if (!reqRecord) return { depth: 0 };

  const current = asObjectRecord(reqRecord.__choysumServiceState);
  if (current) {
    if (typeof current.depth !== 'number') current.depth = 0;
    return current as ServiceState;
  }

  const state: ServiceState = { depth: 0 };
  reqRecord.__choysumServiceState = state;
  return state;
}

function incServiceDepth(req: unknown): void {
  if (!req) return;
  const state = getOrInitServiceState(req);
  state.depth = typeof state.depth === 'number' ? state.depth + 1 : 1;
}

function decServiceDepth(req: unknown): void {
  const reqRecord = asObjectRecord(req);
  const state = asObjectRecord(reqRecord?.__choysumServiceState);
  if (!state) return;
  const depth = state.depth;
  if (typeof depth !== 'number') return;
  state.depth = Math.max(0, depth - 1);
}

function isTopLevelGrpcRequestForReq(req: unknown): boolean {
  const reqRecord = asObjectRecord(req);
  const context = asObjectRecord(reqRecord?.context);
  const reqContext = asObjectRecord(context?.req);
  const kind = reqContext?.kind;
  const isGrpc = kind === 'grpc' || kind === 'grpc-web';

  // IMPORTANT:
  // - jsCtx.req.depth is the trusted *entry depth* injected by Go (0 for top-level inbound).
  // - __choysumServiceState.depth is the in-process nesting depth for conventional service calls (1 for the first service call).
  // We must not overwrite jsCtx.req.depth, otherwise entry allowlists (depth=0) will never trigger.
  const serviceState = asObjectRecord(reqRecord?.__choysumServiceState);
  const callDepth = serviceState?.depth;
  if (typeof callDepth === 'number') {
    return isGrpc && callDepth === 1;
  }
  const entryDepth = reqContext?.depth;
  return isGrpc && entryDepth === 0;
}

export function isTopLevelGrpcRequest(): boolean {
  try {
    const req = getCurrentChoysumRequest();
    return isTopLevelGrpcRequestForReq(req);
  } catch {
    return false;
  }
}

function convertToPlainObject(input: unknown, seen: WeakMap<object, unknown> = new WeakMap()): unknown {
  if (input == null) return input;

  const inputRecord = asObjectRecord(input);

  // Top-level plain payloads (marked by BaseModel) can be returned as-is to avoid a second deep copy or normalization pass.
  if (inputRecord?.__choysum_plain === true) {
    return input;
  }

  if (isDecimal(input)) {
    try {
      return { $bigdecimal: input.toString() };
    } catch {
      return String(input);
    }
  }

  if (typeof input === 'bigint') {
    return { $bigint: input.toString() };
  }

  if (typeof input !== 'object' || !input) return input;

  if (isDecimalLeak(input) && inputRecord) {
    try {
      const reconstructed = new Decimal(0);
      const reconstructedLike = reconstructed as unknown as { s: number; e: number; d: number[] };
      const s = inputRecord.s;
      const e = inputRecord.e;
      const d = inputRecord.d;
      if (typeof s === 'number' && typeof e === 'number' && Array.isArray(d)) {
        reconstructedLike.s = s;
        reconstructedLike.e = e;
        reconstructedLike.d = d.filter((x): x is number => typeof x === 'number');
      }
      return { $bigdecimal: reconstructed.toString() };
    } catch {
      // fall through
    }
  }

  if (input instanceof Date) return input.toISOString();

  if (input instanceof Error) {
    const errOut: ObjectRecord = { name: input.name, message: input.message };
    const inputObj = asObjectRecord(input);
    if (inputObj && hasOwnKey(inputObj, 'code')) errOut.code = inputObj.code;
    return errOut;
  }

  if (input instanceof Map) {
    const out: ObjectRecord = {};
    for (const [k, v] of input.entries()) out[String(k)] = convertToPlainObject(v, seen);
    return out;
  }

  if (input instanceof Set) {
    const out: unknown[] = [];
    for (const v of input.values()) out.push(convertToPlainObject(v, seen));
    return out;
  }

  if (seen.has(input)) return seen.get(input);

  if (Array.isArray(input)) {
    const out: unknown[] = [];
    seen.set(input, out);
    for (const item of input) out.push(convertToPlainObject(item, seen));
    return out;
  }

  // Model instances prefer transport serialization (entity plan + normalization); the result is still traversed recursively, though __choysum_plain usually short-circuits it.
  if (inputRecord && typeof inputRecord.toTransportObject === 'function') {
    try {
      const plain = inputRecord.toTransportObject();
      return convertToPlainObject(plain, seen);
    } catch {
      // fall through
    }
  }

  // Model instances fall back to toPlainObject, but its result still needs recursive traversal.
  if (inputRecord && typeof inputRecord.toPlainObject === 'function') {
    try {
      const plain = inputRecord.toPlainObject();
      return convertToPlainObject(plain, seen);
    } catch {
      // Fall back to the generic object branch.
    }
  }

  // Generic objects.
  const out: ObjectRecord = {};
  seen.set(input, out);

  const descs = Object.getOwnPropertyDescriptors(input);
  for (const key of Object.keys(descs)) {
    const d = descs[key];
    if ('get' in d || 'set' in d) continue;
    const v = d.value;
    if (typeof v === 'function') continue;
    out[key] = convertToPlainObject(v, seen);
  }
  return out;
}

function isBaseModelLike(input: unknown): input is BaseModel {
  const inputRecord = asObjectRecord(input);
  if (!inputRecord) return false;
  // We avoid `instanceof BaseModel` because model instances may be proxies.
  const toPlainObject = inputRecord.toPlainObject;
  if (typeof toPlainObject !== 'function') return false;
  return isModelCtorLike(inputRecord.constructor);
}

function isModelCtorLike(input: unknown): boolean {
  if (!(typeof input === 'function' || typeof input === 'object') || !input) return false;
  // Duck-type via author-surface markers (getRepository was removed from BaseModel).
  const candidate = input as { hydrate?: unknown; Search?: unknown; Browse?: unknown };
  return typeof candidate.hydrate === 'function' || typeof candidate.Search === 'function' || typeof candidate.Browse === 'function';
}

function isChoysumPlainPayload(input: unknown): boolean {
  const inputRecord = asObjectRecord(input);
  return Boolean(inputRecord && inputRecord.__choysum_plain === true);
}

function markPlain<T>(val: T): T {
  if (val && typeof val === 'object') {
    try {
      Object.defineProperty(val as object, '__choysum_plain', {
        value: true,
        enumerable: false,
        configurable: false,
        writable: false,
      });
    } catch {}
  }
  return val as T;
}

async function stripDenyReadFieldsForModelPlain(modelCtor: unknown, plain: unknown, cache: Map<unknown, string[]>): Promise<void> {
  const seen = new WeakSet<object>();

  const resolveRepo = (ctor: unknown): { getDenyReadFields: () => Promise<unknown> } | undefined => {
    if (!(typeof ctor === 'function' || typeof ctor === 'object') || !ctor) return undefined;

    try {
      const repo = RepositoryFactory.getRepository(ctor as Parameters<typeof RepositoryFactory.getRepository>[0]);
      const repoRecord = asObjectRecord(repo);
      if (repoRecord && typeof repoRecord.getDenyReadFields === 'function') {
        return repo as { getDenyReadFields: () => Promise<unknown> };
      }
    } catch {
      return undefined;
    }
    return undefined;
  };

  const getDenyReadFields = async (ctor: unknown): Promise<string[]> => {
    if (!ctor) return [];
    const cached = cache.get(ctor);
    if (cached !== undefined) return cached;

    let denyReadFields: string[] = [];
    try {
      const repo = resolveRepo(ctor);
      if (!repo) {
        cache.set(ctor, denyReadFields);
        return denyReadFields;
      }
      const spec = await repo.getDenyReadFields();
      const specRecord = asObjectRecord(spec);
      denyReadFields = Array.isArray(specRecord?.denyReadFields) ? specRecord.denyReadFields.filter((field): field is string => typeof field === 'string') : [];
    } catch {
      denyReadFields = [];
    }

    cache.set(ctor, denyReadFields);
    return denyReadFields;
  };

  const stripDeep = async (ctor: unknown, obj: unknown): Promise<void> => {
    if (!obj || typeof obj !== 'object') return;
    if (Array.isArray(obj)) {
      for (const it of obj) await stripDeep(ctor, it);
      return;
    }

    const objRecord = asObjectRecord(obj);
    if (!objRecord) return;

    if (seen.has(obj)) return;
    seen.add(obj);

    // 1) Current-layer keys (including relation field keys)
    const denyReadFields = await getDenyReadFields(ctor);
    if (denyReadFields.length) {
      for (const f of denyReadFields) {
        const key = String(f ?? '').trim();
        if (!key) continue;

        // Delete the field value itself, accepting PascalCase, camelCase, and snake_case keys.
        for (const k of buildKeyCandidates(key)) {
          if (hasOwnKey(objRecord, k)) {
            try {
              delete objRecord[k];
            } catch {}
          }
        }

        // Delete relation payload aliases, accepting forms such as $rel$UserId, $rel$user_id, and $rel$_user_id.
        for (const rk of buildRelationAliasCandidates(key)) {
          if (hasOwnKey(objRecord, rk)) {
            try {
              delete objRecord[rk];
            } catch {}
          }
        }
      }
    }

    // 2) Recurse into relations only when model can be determined via metadata.
    //    This avoids accidentally stripping arbitrary business plain objects.
    let meta: unknown;
    try {
      meta = MetadataStorage.instance.getModelMetadata(ctor as Parameters<typeof MetadataStorage.instance.getModelMetadata>[0]);
    } catch {
      meta = undefined;
    }
    const metaRecord = asObjectRecord(meta);
    const fields = metaRecord?.fields;
    if (!(fields instanceof Map)) return;

    for (const [fieldName, fm] of fields as Map<string, unknown>) {
      const fieldMeta = asObjectRecord(fm);
      const rel = asObjectRecord(fieldMeta?.relation);
      const targetModelFn = rel?.targetModel;
      if (typeof targetModelFn !== 'function') continue;

      let targetCtor: unknown;
      try {
        targetCtor = targetModelFn();
      } catch {
        targetCtor = undefined;
      }
      if (!targetCtor) continue;

      // Relation payload may appear under either:
      // - materialized field key (e.g. `UserId`)
      // - raw SQL alias column key (e.g. `$rel$_user_id`)
      const relPayloadKey =
        buildKeyCandidates(fieldName).find(k => hasOwnKey(objRecord, k)) ?? buildRelationAliasCandidates(fieldName).find(k => hasOwnKey(objRecord, k));
      if (!relPayloadKey) continue;

      const rawChild = objRecord[relPayloadKey];
      const child = parseMaybeJsonObject(rawChild);
      if (child !== rawChild) {
        try {
          objRecord[relPayloadKey] = child;
        } catch {}
      }
      if (!child || typeof child !== 'object') continue;
      await stripDeep(targetCtor, child);
    }
  };

  await stripDeep(modelCtor, plain);
}

async function stripDenyReadFieldsForModelPayload(modelCtor: unknown, payload: unknown, cache: Map<unknown, string[]>): Promise<void> {
  if (!payload || typeof payload !== 'object') return;
  if (Array.isArray(payload)) {
    for (const it of payload) {
      await stripDenyReadFieldsForModelPlain(modelCtor, it, cache);
    }
    return;
  }
  await stripDenyReadFieldsForModelPlain(modelCtor, payload, cache);
}

async function maybeConvertAndStripTopLevelModelResult(
  result: unknown,
  denyReadFieldsCache: Map<unknown, string[]>,
  fallbackModelCtor?: unknown
): Promise<unknown> {
  if (isBaseModelLike(result)) {
    const plain = convertToPlainObject(result);
    const modelCtor = asObjectRecord(result)?.constructor || fallbackModelCtor;
    await stripDenyReadFieldsForModelPlain(modelCtor, plain, denyReadFieldsCache);
    return markPlain(plain);
  }

  if (Array.isArray(result) && result.length > 0 && result.every(isBaseModelLike)) {
    const models = result as BaseModel[];
    const out: unknown[] = [];
    for (let i = 0; i < models.length; i++) {
      const it = models[i];
      const plain = convertToPlainObject(it);
      const modelCtor = asObjectRecord(it)?.constructor || fallbackModelCtor;
      await stripDenyReadFieldsForModelPlain(modelCtor, plain, denyReadFieldsCache);
      out.push(plain);
    }
    return markPlain(out);
  }

  return convertToPlainObject(result);
}

function isUppercaseMethodName(name: string): boolean {
  const first = String(name || '').charAt(0);
  if (!first) return false;
  return first === first.toUpperCase() && first !== first.toLowerCase();
}

function isAsyncFunction(value: unknown): value is (...args: unknown[]) => Promise<unknown> {
  if (typeof value !== 'function') return false;
  if (value.constructor?.name === 'AsyncFunction') return true;
  return Object.prototype.toString.call(value) === '[object AsyncFunction]';
}

function getWrappedMethodSet(modelCtor: Function): Set<string> {
  const existing = Reflect.get(modelCtor, WRAPPED_CONVENTIONAL_SERVICE_METHODS);
  if (existing instanceof Set) return existing as Set<string>;

  const wrappedMethods = new Set<string>();
  Object.defineProperty(modelCtor, WRAPPED_CONVENTIONAL_SERVICE_METHODS, {
    value: wrappedMethods,
    enumerable: false,
    configurable: false,
    writable: false,
  });
  return wrappedMethods;
}

function collectConventionalServiceMethods(modelCtor: Function): Array<{ name: string; descriptor: PropertyDescriptor }> {
  const out: Array<{ name: string; descriptor: PropertyDescriptor }> = [];
  const seen = new Set<string>();

  let current: unknown = modelCtor;
  while (typeof current === 'function' && current !== Function.prototype) {
    for (const key of Object.getOwnPropertyNames(current)) {
      if (seen.has(key)) continue;
      seen.add(key);

      if (key === 'length' || key === 'name' || key === 'prototype') continue;
      if (!isUppercaseMethodName(key)) continue;

      const descriptor = Object.getOwnPropertyDescriptor(current, key);
      if (!descriptor || typeof descriptor.value !== 'function') continue;
      if (!isAsyncFunction(descriptor.value)) continue;

      out.push({ name: key, descriptor });
    }
    current = Object.getPrototypeOf(current);
  }

  return out;
}

function assertValidServiceWrapperThisArg(thisArg: unknown, methodName: string): void {
  const kind = getProxyKind(thisArg);
  if (!kind) return;
  throw new Error(`SERVICE_WRAPPER_INVALID_THIS: conventional service "${methodName}" refused draft/proxy thisArg (kind=${kind})`);
}

function invokeWithServiceRuntime(
  thisArg: unknown,
  modelCtor: unknown,
  methodName: string,
  original: (...args: unknown[]) => unknown,
  args: unknown[]
): unknown {
  assertValidServiceWrapperThisArg(thisArg, methodName);
  const req = getCurrentChoysumRequest();
  incServiceDepth(req);
  const topLevelGrpc = isTopLevelGrpcRequestForReq(req);
  const effectiveModelCtor = isModelCtorLike(thisArg) ? thisArg : modelCtor;

  const finalize = (result: unknown) => {
    if (!topLevelGrpc) {
      try {
        return result;
      } finally {
        decServiceDepth(req);
      }
    }

    return (async () => {
      const denyReadFieldsCache = new Map<unknown, string[]>();
      try {
        const out = await maybeConvertAndStripTopLevelModelResult(result, denyReadFieldsCache, effectiveModelCtor);

        if (isChoysumPlainPayload(out) && effectiveModelCtor) {
          await stripDenyReadFieldsForModelPayload(effectiveModelCtor, out, denyReadFieldsCache);
        }

        return markPlain(out);
      } finally {
        decServiceDepth(req);
      }
    })();
  };

  try {
    const ret: unknown = original.apply(thisArg, args);
    const thenFn = (ret && (typeof ret === 'object' || typeof ret === 'function') ? (ret as PromiseLike<unknown>).then : undefined) as
      | ((onfulfilled?: ((value: unknown) => unknown) | null, onrejected?: ((reason: unknown) => unknown) | null) => unknown)
      | undefined;
    if (typeof thenFn === 'function') {
      return (ret as Promise<unknown>).then(
        v => finalize(v),
        err => {
          decServiceDepth(req);
          throw err;
        }
      );
    }
    return finalize(ret);
  } catch (err) {
    decServiceDepth(req);
    throw err;
  }
}

export function installConventionalServiceRuntimeWrappers(modelCtor: unknown): void {
  if (!modelCtor || typeof modelCtor !== 'function') return;
  const ctor = modelCtor as Function;

  const wrappedMethods = getWrappedMethodSet(ctor);
  const candidates = collectConventionalServiceMethods(ctor);

  for (const candidate of candidates) {
    const { name, descriptor } = candidate;
    if (wrappedMethods.has(name)) continue;

    const original = descriptor.value as (...args: unknown[]) => unknown;
    const wrapped = function (this: unknown, ...args: unknown[]) {
      return invokeWithServiceRuntime(this, ctor, name, original, args);
    };

    Object.defineProperty(ctor, name, {
      value: wrapped,
      writable: true,
      configurable: true,
      enumerable: Boolean(descriptor.enumerable),
    });

    wrappedMethods.add(name);
  }
}
