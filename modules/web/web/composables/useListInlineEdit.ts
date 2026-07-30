// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * S2 list inline edit state: one row draft, explicit Save / Discard, no blur RPC.
 */

import { computed, provide, ref, type Ref } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import type { BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { provideOnchange, useProvidedOnchange } from '@/web/web/composables/useOnchange';
import {
  cloneRowDraft,
  collectRowDirtyPayload,
  isListRecordRow,
  isRowDraftDirty,
  listRecordId,
  setDraftField,
  getDraftField,
  unwrapListRecord,
  withEditingPayload,
} from '@/web/web/composables/listRowEdit';
import { createTranslate } from '@/web/web/i18n';

export function useListInlineEdit<T extends BaseModel>(opts: {
  store: WebModelStore<T>;
  enabled: Ref<boolean>;
  listViewMode: Ref<ViewMode>;
  translateScope?: string;
  onSaved?: () => void | Promise<void>;
}) {
  const { _t } = createTranslate('web', { scope: opts.translateScope ?? 'web/composables/useListInlineEdit' });

  const editingRowId = ref<string | null>(null);
  const editingDraft = ref<Record<string, any> | null>(null);
  const editingOriginal = ref<Record<string, any> | null>(null);
  const saving = ref(false);

  const isEditing = computed(() => editingRowId.value != null);

  provide('list-editing-row-id', editingRowId);

  // Always provide form-root so inject is stable; draft is null until a row is being edited (D4).
  provide('form-root', {
    get draft() {
      return opts.enabled.value ? editingDraft.value : null;
    },
    getField(path: string) {
      if (!opts.enabled.value || !editingDraft.value) return undefined;
      return getDraftField(editingDraft.value, path);
    },
    setField(path: string, value: any) {
      if (!opts.enabled.value) return;
      if (!editingDraft.value) editingDraft.value = {};
      setDraftField(editingDraft.value, path, value);
    },
  });

  provideOnchange(opts.store, 'ListView', {
    getRoot: () => (opts.enabled.value ? editingDraft.value ?? undefined : undefined),
    onPatch: value => {
      if (!opts.enabled.value) return;
      if (editingDraft.value && value && typeof value === 'object') {
        Object.assign(editingDraft.value, value);
      }
    },
  });

  function mapItemsWithDraft(items: any[]): any[] {
    if (!isEditing.value || !editingDraft.value) return items;
    return items.map(row => withEditingPayload(row, editingRowId.value, editingDraft.value));
  }

  function isDirty(): boolean {
    return isRowDraftDirty(editingOriginal.value, editingDraft.value, opts.store.fieldsMetadata as any);
  }

  async function promptDirtySwitch(): Promise<'save' | 'discard' | 'cancel'> {
    try {
      await ElMessageBox.confirm(
        _t('You have unsaved changes on the current row. Save before switching?'),
        _t('Unsaved changes'),
        {
          distinguishCancelAndClose: true,
          confirmButtonText: _t('Save'),
          cancelButtonText: _t('Discard'),
          type: 'warning',
        }
      );
      return 'save';
    } catch (action) {
      if (action === 'cancel') return 'discard';
      return 'cancel';
    }
  }

  function exitEdit() {
    editingRowId.value = null;
    editingDraft.value = null;
    editingOriginal.value = null;
    opts.listViewMode.value = 'display';
    useProvidedOnchange()?.reset();
  }

  async function discard() {
    exitEdit();
  }

  async function save(): Promise<boolean> {
    if (!editingDraft.value || !editingOriginal.value || !editingRowId.value) return false;
    saving.value = true;
    try {
      await useProvidedOnchange()?.flush();
      // Onchange flush may clear the draft; treat that as a no-op save.
      if (!editingDraft.value || !editingOriginal.value || !editingRowId.value) return false;
      const payload = collectRowDirtyPayload(editingOriginal.value, editingDraft.value, opts.store.fieldsMetadata as any);
      if (Object.keys(payload).length === 0) {
        exitEdit();
        try {
          await opts.onSaved?.();
        } catch {
          /* persist skipped; refresh failure is non-fatal */
        }
        return true;
      }
      await opts.store.UpdateById(editingRowId.value, payload as any);
      ElMessage.success(_t('Row saved'));
      exitEdit();
      try {
        await opts.onSaved?.();
      } catch {
        /* Update already succeeded; do not surface as save failure */
      }
      return true;
    } catch (e: any) {
      ElMessage.error(_t('Failed to save row'));
      throw e;
    } finally {
      saving.value = false;
    }
  }

  async function enterEdit(row: any): Promise<boolean> {
    if (!opts.enabled.value) return false;
    if (!isListRecordRow(row)) return false;

    const record = unwrapListRecord(row);
    const id = listRecordId(record);
    if (!id) return false;

    if (isEditing.value && editingRowId.value === id) return true;

    if (isEditing.value && isDirty()) {
      const choice = await promptDirtySwitch();
      if (choice === 'cancel') return false;
      if (choice === 'save') {
        const ok = await save();
        if (!ok) return false;
      } else {
        exitEdit();
      }
    } else if (isEditing.value) {
      exitEdit();
    }

    editingOriginal.value = cloneRowDraft(record);
    editingDraft.value = cloneRowDraft(record);
    editingRowId.value = id;
    opts.listViewMode.value = 'edit';
    useProvidedOnchange()?.reset();
    return true;
  }

  return {
    editingRowId,
    editingDraft,
    isEditing,
    saving,
    isDirty,
    enterEdit,
    save,
    discard,
    mapItemsWithDraft,
  };
}
