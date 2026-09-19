// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import { withModelSudo } from './model_sudo';
import { lookupPropertyDefinitionModel } from './properties_lookup';
import { withPropertyDefinitionParentAclBypass } from './properties_definition_acl';
import type { ModelCtor } from './types';
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
  if (!Ctor || typeof Ctor.Search !== 'function') {
    return 0;
  }
  const canBulkDelete = typeof Ctor.Delete === 'function';
  const canDeleteById = typeof Ctor.DeleteById === 'function';
  if (!canBulkDelete && !canDeleteById) return 0;

  // Match both short and application-qualified ContainerModel spellings.
  const modelNames = [...new Set([model, `${app}.${model}`])];
  const condition = {
    And: [
      ['ContainerModel', 'in', modelNames],
      ['ContainerId', 'in', ids],
    ],
  };

  return await withModelSudo(async () => {
    return await withPropertyDefinitionParentAclBypass(async () => {
      if (canBulkDelete && Ctor.Delete) {
        return Number(await Ctor.Delete(condition)) || 0;
      }
      const rows = await Ctor.Search(condition, { fields: ['Id'] });
      let n = 0;
      for (const row of rows || []) {
        const id = String(row?.Id || '').trim();
        if (!id || !Ctor.DeleteById) continue;
        n += Number(await Ctor.DeleteById(id)) || 0;
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
  ModelCtor: ModelCtor,
  deletedIds: string[]
): Promise<void> {
  const ids = [...new Set((deletedIds || []).map(id => String(id || '').trim()).filter(Boolean))];
  if (!ids.length) return;

  const meta = MetadataStorage.instance.getModelMetadata(ModelCtor);
  const application = String(meta?.application || '').trim();
  const containerModel = String(meta?.modelName || meta?.name || '').trim();
  if (!application || !containerModel) return;

  // Never recurse into PropertyDefinition self-deletes as a "parent".
  if (containerModel === 'PropertyDefinition') return;

  await purgePropertyDefinitionsForContainers(application, containerModel, ids);
}
