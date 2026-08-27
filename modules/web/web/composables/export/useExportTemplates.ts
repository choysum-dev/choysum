// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref } from 'vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import { actorUserId } from '@/web/web/composables/search/actorUserId';
import { resolveUserFilterUserId } from '@/web/web/composables/search/userFilterDefaults';

export type ExportTemplateRow = {
  Id?: string;
  Name?: string;
  Application?: string;
  ModelName?: string;
  Fields?: string[];
  ImportCompatible?: boolean;
  UserId?: string | null | { Id?: string | null };
  CreatedUid?: string | null;
};

export type ExportTemplateItem = ExportTemplateRow & {
  Id: string;
  Name: string;
  shared: boolean;
  createUid: string;
  canDelete: boolean;
};

export function parseExportModelRef(model: string): { application: string; modelName: string } {
  const raw = String(model || '').trim();
  const idx = raw.indexOf('.');
  if (idx <= 0 || idx >= raw.length - 1) {
    return { application: '', modelName: '' };
  }
  return { application: raw.slice(0, idx), modelName: raw.slice(idx + 1) };
}

/**
 * Load / apply / save / remove web.ExportTemplate rows for one target model.
 */
export function useExportTemplates(modelRef: () => string) {
  const templates = ref<ExportTemplateItem[]>([]);
  const loading = ref(false);
  const loadError = ref<string | null>(null);
  let loadGeneration = 0;

  const target = computed(() => parseExportModelRef(modelRef()));

  function exportTemplateStore() {
    return createStoreByModel('web.ExportTemplate');
  }

  async function load(): Promise<void> {
    const gen = ++loadGeneration;
    const { application, modelName } = target.value;
    if (!application || !modelName) {
      await Promise.resolve();
      if (gen === loadGeneration) {
        templates.value = [];
        loading.value = false;
        loadError.value = null;
      }
      return;
    }
    loading.value = true;
    loadError.value = null;
    try {
      const me = actorUserId();
      const store = exportTemplateStore() as any;
      const rows = (await store.Search(
        {
          And: [
            ['Application', '=', application],
            ['ModelName', '=', modelName],
            {
              Or: me
                ? [
                    ['UserId', '=', me],
                    ['UserId', '=', null],
                  ]
                : [['UserId', '=', null]],
            },
          ],
        },
        {
          fields: ['Id', 'Name', 'Fields', 'ImportCompatible', 'UserId', 'CreatedUid'],
          orderBy: { field: 'Name', order: 'asc' },
        }
      )) as ExportTemplateRow[];

      if (gen !== loadGeneration) return;

      templates.value = (rows || [])
        .filter(r => r && r.Id && r.Name)
        .map(r => {
          const createUid = String(r.CreatedUid || '').trim();
          const ownerId = resolveUserFilterUserId(r.UserId);
          const shared = !ownerId;
          return {
            ...r,
            Id: String(r.Id),
            Name: String(r.Name),
            Fields: Array.isArray(r.Fields) ? r.Fields.map(String) : [],
            shared,
            createUid,
            canDelete: shared ? !!me && createUid === me : !!me && ownerId === me,
          };
        });
    } catch (e: any) {
      if (gen !== loadGeneration) return;
      loadError.value = e instanceof Error ? e.message : String(e);
      templates.value = [];
    } finally {
      if (gen === loadGeneration) {
        loading.value = false;
      }
    }
  }

  function apply(template: Pick<ExportTemplateRow, 'Fields'>): string[] {
    return (template.Fields ?? []).map(String).filter(Boolean);
  }

  async function saveCurrent(opts: {
    name: string;
    shared?: boolean;
    fields: string[];
    importCompatible?: boolean;
  }): Promise<ExportTemplateItem | null> {
    const { application, modelName } = target.value;
    const name = String(opts.name || '').trim();
    const fields = (opts.fields ?? []).map(String).filter(Boolean);
    if (!application || !modelName || !name || fields.length === 0) return null;

    const me = actorUserId();
    const store = exportTemplateStore() as any;
    const values: Record<string, unknown> = {
      Name: name,
      Application: application,
      ModelName: modelName,
      Fields: fields,
      ImportCompatible: !!opts.importCompatible,
    };
    if (opts.shared) {
      values.UserId = null;
    } else if (me) {
      values.UserId = me;
    }
    const created = await store.Create(values, [
      'Id',
      'Name',
      'Fields',
      'ImportCompatible',
      'UserId',
      'CreatedUid',
    ]);
    await load();
    const createUid = String((created as ExportTemplateRow)?.CreatedUid || me || '').trim();
    const shared = !resolveUserFilterUserId((created as any).UserId);
    return {
      ...(created as ExportTemplateRow),
      Id: String((created as any).Id),
      Name: String((created as any).Name || name),
      Fields: fields,
      shared,
      createUid,
      canDelete: true,
    };
  }

  async function remove(id: string): Promise<void> {
    const store = exportTemplateStore() as any;
    await store.DeleteById(String(id));
    await load();
  }

  return {
    templates,
    loading,
    loadError,
    load,
    apply,
    saveCurrent,
    remove,
  } as const;
}
