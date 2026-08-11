// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { raiseDomainError } from '@/core/service/error';
import { getCurrentReq, getOrInitReqServiceState } from '../../runtime/context';
import { MetadataStorage } from '../metadata/storage';
import { RepositoryFactory } from '../repository/repository_factory';
import { resolveModelConstructor } from './model_registry';
import type { InstantiableModelCtor } from './types';
import type PropertyDefinitionBaseModel from './property_definition_base_model';

function fail(code: string, message: string): never {
  raiseDomainError('core', code, message);
}

function nullScope(value: unknown): string | null {
  if (value == null) return null;
  const s = String(value).trim();
  return s || null;
}

/** Canonical short model name for ContainerModel (strip optional `app.` prefix). */
export function normalizeContainerModelName(value: unknown): string | null {
  const raw = nullScope(value);
  if (!raw) return null;
  const dot = raw.lastIndexOf('.');
  return dot >= 0 ? raw.slice(dot + 1) || null : raw;
}

type ParentWritableProbe = (
  parentCtor: { Search: (...args: any[]) => Promise<any[]>; new (...args: any[]): unknown },
  containerId: string
) => Promise<void>;

type ParentAclState = {
  propertyDefinitionParentAclBypassDepth?: number;
};

let parentWritableProbeOverride: ParentWritableProbe | undefined;
/** Fallback when no request service state exists (unit harness / scripts). */
const processParentAclState: ParentAclState = {};

function getParentAclState(): ParentAclState {
  const reqState = getOrInitReqServiceState(getCurrentReq()) as ParentAclState | undefined | null;
  return reqState || processParentAclState;
}

function getParentAclBypassDepth(): number {
  const value = getParentAclState().propertyDefinitionParentAclBypassDepth;
  return typeof value === 'number' && Number.isFinite(value) ? value : 0;
}

function restoreParentAclBypassDepth(state: ParentAclState): void {
  const current =
    typeof state.propertyDefinitionParentAclBypassDepth === 'number' &&
    Number.isFinite(state.propertyDefinitionParentAclBypassDepth)
      ? state.propertyDefinitionParentAclBypassDepth
      : 0;
  if (current > 1) state.propertyDefinitionParentAclBypassDepth = current - 1;
  else delete state.propertyDefinitionParentAclBypassDepth;
}

function isPromiseLike<T = unknown>(value: unknown): value is Promise<T> {
  return !!value && typeof (value as { then?: unknown }).then === 'function';
}

/** Skip PP8 parent-write gate (used by system purge). Supports sync and async `fn`. */
export function withPropertyDefinitionParentAclBypass<T>(fn: () => T): T {
  const state = getParentAclState();
  const previous = getParentAclBypassDepth();
  state.propertyDefinitionParentAclBypassDepth = previous + 1;
  const restore = () => restoreParentAclBypassDepth(state);
  try {
    const out = fn();
    if (isPromiseLike(out)) {
      return Promise.resolve(out).finally(restore) as T;
    }
    restore();
    return out;
  } catch (e) {
    restore();
    throw e;
  }
}

/** Test-only: replace the parent writable probe. Pass undefined to clear. */
export function __setParentWritableProbeForTest(probe: ParentWritableProbe | undefined): void {
  parentWritableProbeOverride = probe;
}

async function defaultParentWritableProbe(
  parentCtor: { Search: (...args: any[]) => Promise<any[]> },
  containerId: string
): Promise<void> {
  const rows = await parentCtor.Search({ And: [['Id', '=', containerId]] } as any, {
    fields: ['Id'] as any,
    limit: 1,
  } as any);
  if (!rows?.length) {
    fail(
      'PROPERTY_DEFINITION_PARENT_MISSING',
      `PropertyDefinition parent container "${containerId}" was not found or is not readable`
    );
  }

  const repo = RepositoryFactory.getRepository(parentCtor as any);
  // Company + write RecordRule — same gates Update uses before mutating.
  await repo.assertCompanyWriteAccessForIds([containerId]);
  await repo.assertRecordRuleTargetsAllowed('write', [containerId]);
}

function remapParentProbeError(err: any, containerModel: string, containerId: string): never {
  const code = String(err?.code || err?.errorCode || '');
  if (code.startsWith('PROPERTY_DEFINITION_PARENT_')) throw err;
  const msg = err instanceof Error ? err.message : String(err);
  if (
    code.includes('record_rule') ||
    code.includes('company') ||
    /permission|denied|record rule/i.test(msg)
  ) {
    fail(
      'PROPERTY_DEFINITION_PARENT_WRITE_DENIED',
      `PropertyDefinition write requires write access on parent ${containerModel}/${containerId}: ${msg}`
    );
  }
  throw err;
}

/**
 * PP8: parent-record containers require write access on the parent row.
 * App-level rows (both container dims empty) skip this probe — Method ACL alone applies.
 */
export async function assertPropertyDefinitionParentWritable(
  defCtor: InstantiableModelCtor<PropertyDefinitionBaseModel>,
  vals: Record<string, unknown>
): Promise<void> {
  if (getParentAclBypassDepth() > 0) return;

  const containerId = nullScope(vals.ContainerId);
  const containerModel = normalizeContainerModelName(vals.ContainerModel);

  // Reject half-populated parent scopes in both directions.
  if (containerId && !containerModel) {
    fail(
      'PROPERTY_DEFINITION_PARENT_SCOPE',
      'PropertyDefinition parent-container rows require ContainerModel when ContainerId is set'
    );
  }
  if (containerModel && !containerId) {
    fail(
      'PROPERTY_DEFINITION_PARENT_SCOPE',
      'PropertyDefinition parent-container rows require ContainerId when ContainerModel is set'
    );
  }
  if (!containerId) return; // App-level

  // Test override short-circuits before model pool lookup.
  if (parentWritableProbeOverride) {
    try {
      await parentWritableProbeOverride(
        { Search: async () => [{ Id: containerId }] } as any,
        containerId
      );
    } catch (err: any) {
      remapParentProbeError(err, containerModel!, containerId);
    }
    return;
  }

  const meta = MetadataStorage.instance.getModelMetadata(defCtor as any);
  const application = String((meta as any)?.application || '').trim();
  const parentCtor =
    (application ? resolveModelConstructor(`${application}.${containerModel}`) : undefined) ||
    resolveModelConstructor(containerModel!);

  if (!parentCtor || typeof (parentCtor as any).Search !== 'function') {
    fail(
      'PROPERTY_DEFINITION_PARENT_MODEL',
      `PropertyDefinition parent model "${containerModel}" is not registered in application "${application || '?'}"`
    );
  }

  try {
    await defaultParentWritableProbe(parentCtor as any, containerId);
  } catch (err: any) {
    remapParentProbeError(err, containerModel!, containerId);
  }
}

/** Resolve scope fields from a definition vals bag or an existing row. */
export function definitionScopeFromVals(vals: Record<string, unknown>): {
  containerModel: string | null;
  containerId: string | null;
} {
  return {
    containerModel: normalizeContainerModelName(vals.ContainerModel),
    containerId: nullScope(vals.ContainerId),
  };
}

export function parentScopeKey(vals: Record<string, unknown>): string {
  const { containerModel, containerId } = definitionScopeFromVals(vals);
  return `${containerModel ?? ''}\0${containerId ?? ''}`;
}

/** Normalize ContainerModel on write vals to the canonical short name. */
export function normalizeDefinitionContainerScopeOnVals(vals: Record<string, unknown> | undefined): void {
  if (!vals || !Object.prototype.hasOwnProperty.call(vals, 'ContainerModel')) return;
  vals.ContainerModel = normalizeContainerModelName(vals.ContainerModel);
}

/** Collect unique parent scopes to probe (current + merged when reparenting). */
export function collectParentScopesToProbe(
  current: Record<string, unknown> | undefined,
  values: Record<string, unknown> | undefined
): Record<string, unknown>[] {
  const merged = { ...(current || {}), ...(values || {}) } as Record<string, unknown>;
  const out: Record<string, unknown>[] = [];
  const seen = new Set<string>();
  const push = (row: Record<string, unknown>) => {
    const key = parentScopeKey(row);
    if (seen.has(key)) return;
    seen.add(key);
    out.push(row);
  };
  push(merged);
  if (
    current &&
    values &&
    (Object.prototype.hasOwnProperty.call(values, 'ContainerModel') ||
      Object.prototype.hasOwnProperty.call(values, 'ContainerId'))
  ) {
    push(current);
  }
  return out;
}
