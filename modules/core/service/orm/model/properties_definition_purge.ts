// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import { withModelSudo } from './model_sudo';
import { lookupPropertyDefinitionModel } from './properties_lookup';
import { withPropertyDefinitionParentAclBypass } from './properties_definition_acl';
import type { RuntimeModelCtor } from './types';

/**
 * Delete PropertyDefinition rows scoped to the given parent containers.
 * Used when parent records are deleted (§3.4). Does not scrub child properties JSON.
 */
export async function purgePropertyDefinitionsForContainers(
  application: string,
  containerModel: string,
  containerIds: string[]
): Promise<number> {
  const app = String(application || '').trim();
  const model = String(containerModel || '').trim();
  const ids = [...new Set((containerIds || []).map(id => String(id || '').trim()).filter(Boolean))];
  if (!app || !model || !ids.length) return 0;

  const Ctor = lookupPropertyDefinitionModel(app);
  if (!Ctor || typeof (Ctor as any).Search !== 'function' || typeof (Ctor as any).DeleteById !== 'function') {
    return 0;
  }

  return await withModelSudo(async () => {
    return await withPropertyDefinitionParentAclBypass(async () => {
      const rows = await (Ctor as any).Search(
        {
          And: [
            ['ContainerModel', '=', model],
            ['ContainerId', 'in', ids],
          ],
        } as any,
        { fields: ['Id'] as any } as any
      );
      let n = 0;
      for (const row of rows || []) {
        const id = String((row as any)?.Id || '').trim();
        if (!id) continue;
        n += Number(await (Ctor as any).DeleteById(id)) || 0;
      }
      return n;
    });
  }, { hint: 'purgePropertyDefinitionsForContainers' });
}

/**
 * After a successful delete of model `M`, purge definition rows where
 * ContainerModel=M and ContainerId ∈ deleted ids (same application).
 */
export async function purgePropertyDefinitionsAfterParentDelete(
  ModelCtor: RuntimeModelCtor,
  deletedIds: string[]
): Promise<void> {
  const ids = [...new Set((deletedIds || []).map(id => String(id || '').trim()).filter(Boolean))];
  if (!ids.length) return;

  const meta = MetadataStorage.instance.getModelMetadata(ModelCtor as any);
  const application = String((meta as any)?.application || '').trim();
  const containerModel = String((meta as any)?.modelName || (meta as any)?.name || '').trim();
  if (!application || !containerModel) return;

  // Never recurse into PropertyDefinition self-deletes as a "parent".
  if (containerModel === 'PropertyDefinition') return;

  try {
    await purgePropertyDefinitionsForContainers(application, containerModel, ids);
  } catch (e) {
    if (typeof console !== 'undefined') {
      console.warn('[Delete] PropertyDefinition container purge failed:', e);
    }
    throw e;
  }
}
