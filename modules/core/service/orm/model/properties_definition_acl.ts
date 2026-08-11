// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { raiseDomainError } from '@/core/service/error';
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

type ParentWritableProbe = (
  parentCtor: { Search: (...args: any[]) => Promise<any[]>; new (...args: any[]): unknown },
  containerId: string
) => Promise<void>;

let parentAclBypassDepth = 0;
let parentWritableProbeOverride: ParentWritableProbe | undefined;

/** Skip PP8 parent-write gate (used by system purge). Supports sync and async `fn`. */
export function withPropertyDefinitionParentAclBypass<T>(fn: () => T): T {
  parentAclBypassDepth += 1;
  try {
    const out = fn();
    if (out && typeof (out as any).then === 'function') {
      return Promise.resolve(out).finally(() => {
        parentAclBypassDepth -= 1;
      }) as T;
    }
    parentAclBypassDepth -= 1;
    return out;
  } catch (e) {
    parentAclBypassDepth -= 1;
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

/**
 * PP8: parent-record containers require write access on the parent row.
 * App-level rows (empty ContainerId) skip this probe — Method ACL alone applies.
 */
export async function assertPropertyDefinitionParentWritable(
  defCtor: InstantiableModelCtor<PropertyDefinitionBaseModel>,
  vals: Record<string, unknown>
): Promise<void> {
  if (parentAclBypassDepth > 0) return;

  const containerId = nullScope(vals.ContainerId);
  const containerModel = nullScope(vals.ContainerModel);
  if (!containerId) return; // App-level

  if (!containerModel) {
    fail(
      'PROPERTY_DEFINITION_PARENT_SCOPE',
      'PropertyDefinition parent-container rows require ContainerModel when ContainerId is set'
    );
  }

  // Test override short-circuits before model pool lookup.
  if (parentWritableProbeOverride) {
    try {
      await parentWritableProbeOverride(
        { Search: async () => [{ Id: containerId }] } as any,
        containerId
      );
    } catch (err: any) {
      const code = String(err?.code || err?.errorCode || '');
      const msg = err instanceof Error ? err.message : String(err);
      if (code.startsWith('PROPERTY_DEFINITION_PARENT_')) throw err;
      fail(
        'PROPERTY_DEFINITION_PARENT_WRITE_DENIED',
        `PropertyDefinition write requires write access on parent ${containerModel}/${containerId}: ${msg}`
      );
    }
    return;
  }

  const meta = MetadataStorage.instance.getModelMetadata(defCtor as any);
  const application = String((meta as any)?.application || '').trim();
  const parentCtor =
    (application ? resolveModelConstructor(`${application}.${containerModel}`) : undefined) ||
    resolveModelConstructor(containerModel);

  if (!parentCtor || typeof (parentCtor as any).Search !== 'function') {
    fail(
      'PROPERTY_DEFINITION_PARENT_MODEL',
      `PropertyDefinition parent model "${containerModel}" is not registered in application "${application || '?'}"`
    );
  }

  try {
    await defaultParentWritableProbe(parentCtor as any, containerId);
  } catch (err: any) {
    const code = String(err?.code || err?.errorCode || '');
    const msg = err instanceof Error ? err.message : String(err);
    if (
      code.includes('PROPERTY_DEFINITION_PARENT_') ||
      code.includes('record_rule') ||
      code.includes('company') ||
      /permission|denied|record rule/i.test(msg)
    ) {
      if (code.startsWith('PROPERTY_DEFINITION_PARENT_')) throw err;
      fail(
        'PROPERTY_DEFINITION_PARENT_WRITE_DENIED',
        `PropertyDefinition write requires write access on parent ${containerModel}/${containerId}: ${msg}`
      );
    }
    throw err;
  }
}

/** Resolve scope fields from a definition vals bag or an existing row. */
export function definitionScopeFromVals(vals: Record<string, unknown>): {
  containerModel: string | null;
  containerId: string | null;
} {
  return {
    containerModel: nullScope(vals.ContainerModel),
    containerId: nullScope(vals.ContainerId),
  };
}
