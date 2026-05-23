// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { asObjectRecord } from './object';
import type { ObjectRecord } from './types';

type RuntimeEnv = ObjectRecord;
const RUNTIME_ENV_GLOBAL_KEYS = ['__CHOYSUM_RUNTIME_ENV__', '__CHOYSUM_ENV__'] as const;
const RUNTIME_GLOBAL_POOL_KEY = 'pool';
const RUNTIME_ONCHANGE_FLAG_OVERRIDES_KEY = '__CHOYSUM_TEST_ONCHANGE_FLAGS__';
const RUNTIME_COMPUTE_AUDIT_BUCKET_KEY = '__choysumComputeAudit';

type RuntimeIntlApi = Pick<typeof Intl, 'DateTimeFormat'>;
type RuntimeImportMetaEnv = {
  CHOYSUM_APP_NAME?: unknown;
  CHOYSUM_MODULE_NAME?: unknown;
  CHOYSUM_SOFT_DELETE_ENABLED?: unknown;
  CHOYSUM_GRPC_COMPANY_FILTER_ENABLED?: unknown;
  CHOYSUM_SELECTION_TREE_STRICT?: unknown;
  CHOYSUM_ENV?: unknown;
  CHOYSUM_AUTHZ_DECISION_LOG?: unknown;
  CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED?: unknown;
  CHOYSUM_GRPC_FIELD_RULE_ENABLED?: unknown;
  CHOYSUM_GRPC_RECORD_RULE_ENABLED?: unknown;
};
type ImportMetaCarrier = {
  env?: RuntimeImportMetaEnv;
};

export function getRuntimeGlobalRecord(): RuntimeEnv {
  return asObjectRecord(globalThis) ?? {};
}

export function getRuntimeGlobalValue(key: string): unknown {
  return getRuntimeGlobalRecord()[key];
}

export function setRuntimeGlobalValue(key: string, value: unknown): void {
  getRuntimeGlobalRecord()[key] = value;
}

export function getRuntimeGlobalPoolValue(): unknown {
  return getRuntimeGlobalValue(RUNTIME_GLOBAL_POOL_KEY);
}

export function setRuntimeGlobalPoolValue(value: unknown): void {
  setRuntimeGlobalValue(RUNTIME_GLOBAL_POOL_KEY, value);
}

export function getRuntimeOnchangeFlagOverridesValue(): unknown {
  return getRuntimeGlobalValue(RUNTIME_ONCHANGE_FLAG_OVERRIDES_KEY);
}

export function getRuntimeComputeAuditBucketValue(): unknown {
  return getRuntimeGlobalValue(RUNTIME_COMPUTE_AUDIT_BUCKET_KEY);
}

export function setRuntimeComputeAuditBucketValue(value: unknown): void {
  setRuntimeGlobalValue(RUNTIME_COMPUTE_AUDIT_BUCKET_KEY, value);
}

export function getRuntimeIntlApi(): RuntimeIntlApi | undefined {
  const candidate = getRuntimeGlobalValue('Intl');
  const record = asObjectRecord(candidate);
  if (!record || typeof record.DateTimeFormat !== 'function') return undefined;
  return candidate as RuntimeIntlApi;
}

function readRuntimeEnvFromGlobal(root: RuntimeEnv): RuntimeEnv | undefined {
  for (const key of RUNTIME_ENV_GLOBAL_KEYS) {
    const candidate = asObjectRecord(root[key]);
    if (candidate) return candidate;
  }

  return undefined;
}

function getRuntimeImportMetaEnv(): RuntimeImportMetaEnv | undefined {
  const metaCarrier = import.meta as ImportMetaCarrier;
  return metaCarrier.env;
}

function readRuntimeEnvValueFromImportMeta(key: string): unknown {
  const runtimeImportMetaEnv = getRuntimeImportMetaEnv();
  if (!runtimeImportMetaEnv) return undefined;

  switch (key) {
    case 'CHOYSUM_APP_NAME':
      return runtimeImportMetaEnv.CHOYSUM_APP_NAME;
    case 'CHOYSUM_MODULE_NAME':
      return runtimeImportMetaEnv.CHOYSUM_MODULE_NAME;
    case 'CHOYSUM_SOFT_DELETE_ENABLED':
      return runtimeImportMetaEnv.CHOYSUM_SOFT_DELETE_ENABLED;
    case 'CHOYSUM_GRPC_COMPANY_FILTER_ENABLED':
      return runtimeImportMetaEnv.CHOYSUM_GRPC_COMPANY_FILTER_ENABLED;
    case 'CHOYSUM_SELECTION_TREE_STRICT':
      return runtimeImportMetaEnv.CHOYSUM_SELECTION_TREE_STRICT;
    case 'CHOYSUM_ENV':
      return runtimeImportMetaEnv.CHOYSUM_ENV;
    case 'CHOYSUM_AUTHZ_DECISION_LOG':
      return runtimeImportMetaEnv.CHOYSUM_AUTHZ_DECISION_LOG;
    case 'CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED':
      return runtimeImportMetaEnv.CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED;
    case 'CHOYSUM_GRPC_FIELD_RULE_ENABLED':
      return runtimeImportMetaEnv.CHOYSUM_GRPC_FIELD_RULE_ENABLED;
    case 'CHOYSUM_GRPC_RECORD_RULE_ENABLED':
      return runtimeImportMetaEnv.CHOYSUM_GRPC_RECORD_RULE_ENABLED;
    default:
      return undefined;
  }
}

export function getRuntimeEnv(): RuntimeEnv {
  const root = getRuntimeGlobalRecord();
  return {
    ...(readRuntimeEnvFromGlobal(root) ?? {}),
  };
}

export function getRuntimeEnvValue(key: string): unknown {
  const fromImportMeta = readRuntimeEnvValueFromImportMeta(key);
  if (fromImportMeta !== undefined) return fromImportMeta;
  return getRuntimeEnv()[key];
}

// Compatibility mode for feature flags: only explicit "false" disables.
export function resolveRuntimeEnvFlag(raw: unknown, defaultValue: boolean): boolean {
  if (typeof raw === 'boolean') return raw;
  if (typeof raw === 'string') return raw.trim().toLowerCase() !== 'false';
  return defaultValue;
}

export function getRuntimeEnvFlag(key: string, defaultValue: boolean): boolean {
  return resolveRuntimeEnvFlag(getRuntimeEnvValue(key), defaultValue);
}

export function parseRuntimeEnvBoolean(raw: unknown): boolean | undefined {
  if (typeof raw === 'boolean') return raw;
  if (typeof raw !== 'string') return undefined;

  const normalized = raw.trim().toLowerCase();
  if (!normalized) return undefined;
  if (normalized === 'true' || normalized === '1' || normalized === 'yes' || normalized === 'on') return true;
  if (normalized === 'false' || normalized === '0' || normalized === 'no' || normalized === 'off') return false;
  return undefined;
}

export function getRuntimeEnvBoolean(key: string): boolean | undefined {
  return parseRuntimeEnvBoolean(getRuntimeEnvValue(key));
}
