// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, inject, provide, ref } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';

import { disposeOnchange } from '@/web/web/composables/useOnchange';
import { useListInlineEdit } from '@/web/web/composables/useListInlineEdit';
import { fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';

const origConfirm = ElMessageBox.confirm;
const origSuccess = ElMessage.success;
const origError = ElMessage.error;

function mountInline(opts?: {
  enabled?: boolean;
  onSaved?: () => void | Promise<void>;
  updateImpl?: (id: string, payload: any) => Promise<any>;
}) {
  const enabled = ref(opts?.enabled ?? true);
  const UpdateById = fnRecorder(opts?.updateImpl ?? (async () => ({})));
  const Onchange = fnRecorder(async () => ({ value: {}, messages: [] }));
  const store = {
    fieldsMetadata: {
      Name: { id: '1', type: 'varchar', typeAnnotation: '' },
      Sequence: { id: '2', type: 'int', typeAnnotation: '', isReadonly: true },
    },
    UpdateById,
    Onchange,
    state: {
      record: { Id: 'store-rec', Name: 'Store' },
      _draftRecord: null as any,
    },
  } as any;

  let api: ReturnType<typeof useListInlineEdit> | null = null;
  let headerFormRoot: any = 'unset';
  let tableFormRoot: any = 'unset';

  const Host = defineComponent({
    setup() {
      api = useListInlineEdit({
        store,
        enabled,
        onSaved: opts?.onSaved,
      });

      const HeaderProbe = defineComponent({
        setup() {
          headerFormRoot = inject('form-root', null);
          return () => h('span', { class: 'header-probe' });
        },
      });

      const TableScope = defineComponent({
        setup(_, { slots }) {
          provide('form-root', api!.formRoot);
          provide('view-mode', api!.tableViewMode);
          return () => h('div', { class: 'table-scope' }, slots.default?.());
        },
      });

      const TableProbe = defineComponent({
        setup() {
          tableFormRoot = inject('form-root', null);
          return () => h('span', { class: 'table-probe' });
        },
      });

      return () => h('div', [h(HeaderProbe), h(TableScope, null, { default: () => h(TableProbe) })]);
    },
  });

  const mounted = mountApp(Host);
  return {
    api: api!,
    enabled,
    store,
    UpdateById,
    Onchange,
    formRoot: () => api!.formRoot,
    headerFormRoot: () => headerFormRoot,
    tableFormRoot: () => tableFormRoot,
    unmount: () => {
      mounted.unmount();
      disposeOnchange(store);
    },
  };
}

describe('useListInlineEdit', () => {
  const confirm = fnRecorder(async () => true as any);
  const success = fnRecorder();
  const error = fnRecorder();

  beforeEach(() => {
    confirm.mockReset();
    confirm.mockImplementation(async () => true);
    success.mockReset();
    error.mockReset();
    (ElMessageBox as any).confirm = confirm;
    (ElMessage as any).success = success;
    (ElMessage as any).error = error;
  });

  afterEach(() => {
    (ElMessageBox as any).confirm = origConfirm;
    (ElMessage as any).success = origSuccess;
    (ElMessage as any).error = origError;
  });

  test('rejects enterEdit when disabled, non-record, or missing id', async () => {
    const { api, enabled, unmount } = mountInline();
    enabled.value = false;
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } })).toBe(false);
    enabled.value = true;
    expect(await api.enterEdit({ kind: 'group' })).toBe(false);
    expect(await api.enterEdit({ kind: 'record', payload: { Name: 'no-id' } })).toBe(false);
    unmount();
  });

  test('enters edit and maps draft onto matching rows', async () => {
    const { api, unmount } = mountInline();
    const row = { kind: 'record', key: '1', payload: { Id: '1', Name: 'A' } };
    expect(await api.enterEdit(row)).toBe(true);
    expect(api.isEditing.value).toBe(true);
    expect(api.tableViewMode.value).toBe('edit');
    expect(api.editingDraft.value?.Name).toBe('A');
    expect(await api.enterEdit(row)).toBe(true);

    api.editingDraft.value!.Name = 'B';
    const mapped = api.mapItemsWithDraft([row, { kind: 'record', key: '2', payload: { Id: '2', Name: 'X' } }]);
    expect(mapped[0].payload.Name).toBe('B');
    expect(mapped[1].payload.Name).toBe('X');
    unmount();
  });

  test('discards draft and resets table view mode', async () => {
    const { api, unmount } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    await api.discard();
    expect(api.isEditing.value).toBe(false);
    expect(api.tableViewMode.value).toBe('display');
    expect(api.mapItemsWithDraft([{ kind: 'record', payload: { Id: '1' } }])[0].payload.Id).toBe('1');
    unmount();
  });

  test('saves dirty payload via UpdateById and swallows onSaved errors', async () => {
    const onSaved = fnRecorder(async () => {
      throw new Error('reload failed');
    });
    const { api, UpdateById, unmount } = mountInline({ onSaved });
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A', Sequence: 1 } });
    api.editingDraft.value!.Name = 'B';
    expect(api.isDirty()).toBe(true);
    await expect(api.save()).resolves.toBe(true);
    expect(UpdateById.calls[0]).toEqual(['1', { Name: 'B' }]);
    expect(success.calls.length).toBeGreaterThan(0);
    expect(onSaved.calls.length).toBe(1);
    expect(api.isEditing.value).toBe(false);
    unmount();
  });

  test('saves with empty dirty payload without UpdateById', async () => {
    const onSaved = fnRecorder(async () => {
      throw new Error('x');
    });
    const { api, UpdateById, unmount } = mountInline({ onSaved });
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    await expect(api.save()).resolves.toBe(true);
    expect(UpdateById.calls.length).toBe(0);
    expect(onSaved.calls.length).toBe(1);
    unmount();
  });

  test('surfaces UpdateById failures', async () => {
    const { api, unmount } = mountInline({
      updateImpl: async () => {
        throw new Error('boom');
      },
    });
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';
    await expect(api.save()).rejects.toThrow('boom');
    expect(error.calls.length).toBeGreaterThan(0);
    expect(api.saving.value).toBe(false);
    unmount();
  });

  test('save returns false when not editing', async () => {
    const { api, unmount } = mountInline();
    expect(await api.save()).toBe(false);
    unmount();
  });

  test('dirty switch save / discard / cancel', async () => {
    const { api, unmount } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';

    confirm.mockImplementation(async () => true);
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '2', Name: 'C' } })).toBe(true);
    expect(api.editingRowId.value).toBe('2');

    await api.enterEdit({ kind: 'record', payload: { Id: '2', Name: 'C' } });
    api.editingDraft.value!.Name = 'D';
    confirm.mockImplementation(async () => {
      throw 'cancel';
    });
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '3', Name: 'E' } })).toBe(true);
    expect(api.editingRowId.value).toBe('3');

    api.editingDraft.value!.Name = 'F';
    confirm.mockImplementation(async () => {
      throw 'close';
    });
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '4', Name: 'G' } })).toBe(false);
    expect(api.editingRowId.value).toBe('3');
    unmount();
  });

  test('dirty switch save failure blocks enter', async () => {
    const { api, unmount } = mountInline({
      updateImpl: async () => {
        throw new Error('fail');
      },
    });
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';
    confirm.mockImplementation(async () => true);
    await expect(api.enterEdit({ kind: 'record', payload: { Id: '2', Name: 'C' } })).rejects.toThrow('fail');
    unmount();
  });

  test('mapItemsWithDraft returns the same array when not editing', () => {
    const { api, unmount } = mountInline();
    const rows = [{ kind: 'record', payload: { Id: '1', Name: 'A' } }];
    expect(api.mapItemsWithDraft(rows)).toBe(rows);
    unmount();
  });

  test('exits cleanly when switching undirty rows', async () => {
    const { api, unmount } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '2', Name: 'B' } })).toBe(true);
    expect(api.editingRowId.value).toBe('2');
    unmount();
  });

  test('does not leak form-root to header siblings outside the table scope', () => {
    const { headerFormRoot, tableFormRoot, formRoot, unmount } = mountInline();
    expect(headerFormRoot()).toBeNull();
    expect(tableFormRoot()).toBe(formRoot());
    unmount();
  });

  test('form-root getField/setField respect enabled and draft state', async () => {
    const { api, enabled, formRoot, unmount } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A', nested: { x: 1 } } });

    const root = formRoot();
    expect(root.getField('Name')).toBe('A');
    expect(root.getField('nested.x')).toBe(1);
    root.setField('Name', 'Z');
    root.setField('nested.y', 2);
    expect(api.editingDraft.value?.Name).toBe('Z');
    expect(api.editingDraft.value?.nested.y).toBe(2);

    enabled.value = false;
    expect(root.draft).toBeNull();
    expect(root.getField('Name')).toBeUndefined();
    root.setField('Name', 'ignored');
    expect(api.editingDraft.value?.Name).toBe('Z');
    unmount();
  });

  test('form-root setField initializes empty draft when missing', async () => {
    const { api, formRoot, unmount } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value = null;
    const root = formRoot();
    expect(root.draft).toBeNull();
    expect(root.getField('Name')).toBeUndefined();
    root.setField('Name', 'bootstrapped');
    expect(api.editingDraft.value).toEqual({ Name: 'bootstrapped' });
    unmount();
  });

  test('save rejects re-entrant calls while a save is in flight', async () => {
    let release!: () => void;
    const gate = new Promise<void>(resolve => {
      release = resolve;
    });
    const { api, UpdateById, unmount } = mountInline({
      updateImpl: async () => {
        await gate;
        return {};
      },
    });
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';
    const first = api.save();
    await Promise.resolve();
    expect(api.saving.value).toBe(true);
    expect(await api.save()).toBe(false);
    release();
    await expect(first).resolves.toBe(true);
    expect(UpdateById.calls.length).toBe(1);
    unmount();
  });
});

// density backfill from main before merge
// Remaining mock-driven flush / onchange-wiring cases need optional DI on provideOnchange.
