// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, inject, provide, ref } from 'vue';
import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const flushMock = vi.fn(async () => {});
const resetMock = vi.fn();
const pauseMock = vi.fn();
const onchangeCtrl = {
  flush: flushMock,
  reset: resetMock,
  pause: pauseMock,
  force: vi.fn(),
  running: ref(false),
};

vi.mock('@/web/web/composables/useOnchange', () => ({
  provideOnchange: vi.fn(() => onchangeCtrl),
  useProvidedOnchange: vi.fn(() => onchangeCtrl),
}));

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  return {
    ...actual,
    ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
    ElMessageBox: { confirm: vi.fn() },
  };
});

import { ElMessage, ElMessageBox } from 'element-plus';
import { useListInlineEdit } from '@/web/web/composables/useListInlineEdit';

function mountInline(opts?: {
  enabled?: boolean;
  onSaved?: () => void | Promise<void>;
  updateImpl?: (id: string, payload: any) => Promise<any>;
}) {
  const enabled = ref(opts?.enabled ?? true);
  const UpdateById = vi.fn(opts?.updateImpl ?? (async () => ({})));
  const store = {
    fieldsMetadata: {
      Name: { id: '1', type: 'varchar', typeAnnotation: '' },
      Sequence: { id: '2', type: 'int', typeAnnotation: '', isReadonly: true },
    },
    UpdateById,
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

      // Sibling of the table scope — must not see form-root from the composable.
      const HeaderProbe = defineComponent({
        setup() {
          headerFormRoot = inject('form-root', null);
          return () => h('span', { class: 'header-probe' });
        },
      });

      // Simulates OListInlineEditScope providing form-root under the table only.
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

  mount(Host);
  return {
    api: api!,
    enabled,
    store,
    UpdateById,
    formRoot: () => api!.formRoot,
    headerFormRoot: () => headerFormRoot,
    tableFormRoot: () => tableFormRoot,
  };
}

describe('useListInlineEdit', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    flushMock.mockResolvedValue(undefined);
  });

  it('rejects enterEdit when disabled, non-record, or missing id', async () => {
    const { api, enabled } = mountInline();
    enabled.value = false;
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } })).toBe(false);
    enabled.value = true;
    expect(await api.enterEdit({ kind: 'group' })).toBe(false);
    expect(await api.enterEdit({ kind: 'record', payload: { Name: 'no-id' } })).toBe(false);
  });

  it('enters edit and maps draft onto matching rows', async () => {
    const { api } = mountInline();
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
  });

  it('discards draft and resets table view mode', async () => {
    const { api } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    await api.discard();
    expect(api.isEditing.value).toBe(false);
    expect(api.tableViewMode.value).toBe('display');
    expect(api.mapItemsWithDraft([{ kind: 'record', payload: { Id: '1' } }])[0].payload.Id).toBe('1');
  });

  it('saves dirty payload via UpdateById and swallows onSaved errors', async () => {
    const onSaved = vi.fn(async () => {
      throw new Error('reload failed');
    });
    const { api, UpdateById } = mountInline({ onSaved });
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A', Sequence: 1 } });
    api.editingDraft.value!.Name = 'B';
    expect(api.isDirty()).toBe(true);
    await expect(api.save()).resolves.toBe(true);
    expect(UpdateById).toHaveBeenCalledWith('1', { Name: 'B' });
    expect(ElMessage.success).toHaveBeenCalled();
    expect(onSaved).toHaveBeenCalled();
    expect(api.isEditing.value).toBe(false);
  });

  it('saves with empty dirty payload without UpdateById', async () => {
    const onSaved = vi.fn(async () => {
      throw new Error('x');
    });
    const { api, UpdateById } = mountInline({ onSaved });
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    await expect(api.save()).resolves.toBe(true);
    expect(UpdateById).not.toHaveBeenCalled();
    expect(onSaved).toHaveBeenCalled();
  });

  it('surfaces UpdateById failures', async () => {
    const { api } = mountInline({
      updateImpl: async () => {
        throw new Error('boom');
      },
    });
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';
    await expect(api.save()).rejects.toThrow('boom');
    expect(ElMessage.error).toHaveBeenCalled();
    expect(api.saving.value).toBe(false);
  });

  it('save returns false when not editing', async () => {
    const { api } = mountInline();
    expect(await api.save()).toBe(false);
  });

  it('dirty switch save / discard / cancel', async () => {
    const { api } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';

    (ElMessageBox.confirm as any).mockResolvedValueOnce(true);
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '2', Name: 'C' } })).toBe(true);
    expect(api.editingRowId.value).toBe('2');

    await api.enterEdit({ kind: 'record', payload: { Id: '2', Name: 'C' } });
    api.editingDraft.value!.Name = 'D';
    (ElMessageBox.confirm as any).mockRejectedValueOnce('cancel');
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '3', Name: 'E' } })).toBe(true);
    expect(api.editingRowId.value).toBe('3');

    api.editingDraft.value!.Name = 'F';
    (ElMessageBox.confirm as any).mockRejectedValueOnce('close');
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '4', Name: 'G' } })).toBe(false);
    expect(api.editingRowId.value).toBe('3');
  });

  it('dirty switch save failure blocks enter', async () => {
    const { api } = mountInline({
      updateImpl: async () => {
        throw new Error('fail');
      },
    });
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';
    (ElMessageBox.confirm as any).mockResolvedValueOnce(true);
    await expect(api.enterEdit({ kind: 'record', payload: { Id: '2', Name: 'C' } })).rejects.toThrow('fail');
  });

  it('dirty switch aborts when save returns false', async () => {
    const { api } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';
    (ElMessageBox.confirm as any).mockImplementationOnce(async () => {
      // Clear draft so save() returns false without throwing.
      api.editingDraft.value = null;
      return true;
    });
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '2', Name: 'C' } })).toBe(false);
    expect(api.editingRowId.value).toBe('1');
  });

  it('mapItemsWithDraft returns the same array when not editing', () => {
    const { api } = mountInline();
    const rows = [{ kind: 'record', payload: { Id: '1', Name: 'A' } }];
    expect(api.mapItemsWithDraft(rows)).toBe(rows);
  });

  it('exitEdit tolerates a missing onchange controller', async () => {
    const { useProvidedOnchange } = await import('@/web/web/composables/useOnchange');
    (useProvidedOnchange as any).mockReturnValueOnce(null);
    const { api } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    (useProvidedOnchange as any).mockReturnValueOnce(null);
    await expect(api.discard()).resolves.toBeUndefined();
    expect(api.isEditing.value).toBe(false);
  });

  it('exits cleanly when switching undirty rows', async () => {
    const { api } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '2', Name: 'B' } })).toBe(true);
    expect(api.editingRowId.value).toBe('2');
  });
});

describe('useListInlineEdit form-root / onchange wiring', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('does not leak form-root to header siblings outside the table scope', async () => {
    const { headerFormRoot, tableFormRoot, formRoot } = mountInline();
    expect(headerFormRoot()).toBeNull();
    expect(tableFormRoot()).toBe(formRoot());
  });

  it('form-root getField/setField respect enabled and draft state', async () => {
    const { api, enabled, formRoot } = mountInline();
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
  });

  it('form-root setField initializes empty draft when missing', async () => {
    const { api, formRoot } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value = null;
    const root = formRoot();
    expect(root.draft).toBeNull();
    expect(root.getField('Name')).toBeUndefined();
    root.setField('Name', 'bootstrapped');
    expect(api.editingDraft.value).toEqual({ Name: 'bootstrapped' });
  });

  it('provideOnchange uses draft root while editing and store fallback when idle', async () => {
    const { provideOnchange } = await import('@/web/web/composables/useOnchange');
    const { api, enabled, store } = mountInline();
    const opts = (provideOnchange as any).mock.calls.at(-1)[2];

    expect(opts.getRoot()).toEqual(expect.objectContaining({ Id: 'store-rec' }));

    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    expect(opts.getRoot()).toEqual(expect.objectContaining({ Id: '1' }));
    opts.onPatch({ Name: 'Z' });
    expect(api.editingDraft.value?.Name).toBe('Z');
    // Truthy non-object must hit the typeof!=='object' branch while a draft exists.
    opts.onPatch('x');
    opts.onPatch(42);
    expect(api.editingDraft.value?.Name).toBe('Z');

    api.editingDraft.value = null;
    expect(opts.getRoot()).toEqual(expect.objectContaining({ Id: 'store-rec' }));
    opts.onPatch({ Name: 'ignored' });
    opts.onPatch(null);
    opts.onPatch('x');
    expect(api.editingDraft.value).toBeNull();

    enabled.value = false;
    expect(opts.getRoot()).toEqual(expect.objectContaining({ Id: 'store-rec' }));
    store.state._draftRecord = { Id: 'draft', Name: 'D' };
    expect(opts.getRoot()).toEqual(expect.objectContaining({ Id: 'draft' }));
  });

  it('re-pauses onchange after reset on enter and exit edit', async () => {
    const { api } = mountInline();
    resetMock.mockClear();
    pauseMock.mockClear();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    expect(resetMock).toHaveBeenCalled();
    expect(pauseMock).toHaveBeenCalled();
    expect(pauseMock.mock.invocationCallOrder[0]).toBeGreaterThan(resetMock.mock.invocationCallOrder[0]!);

    resetMock.mockClear();
    pauseMock.mockClear();
    await api.discard();
    expect(resetMock).toHaveBeenCalled();
    expect(pauseMock).toHaveBeenCalled();
    expect(pauseMock.mock.invocationCallOrder[0]).toBeGreaterThan(resetMock.mock.invocationCallOrder[0]!);
  });

  it('save rejects re-entrant calls while a save is in flight', async () => {
    let release!: () => void;
    const gate = new Promise<void>(resolve => {
      release = resolve;
    });
    const { api, UpdateById } = mountInline({
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
    expect(UpdateById).toHaveBeenCalledTimes(1);
  });

  it('exits edit when flush clears draft during save', async () => {
    const { api } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';
    flushMock.mockImplementationOnce(async () => {
      api.editingDraft.value = null;
    });
    await expect(api.save()).resolves.toBe(true);
    expect(api.isEditing.value).toBe(false);
    expect(api.editingRowId.value).toBeNull();
  });

  it('dirty switch continues after flush-cleared draft save exits edit', async () => {
    const { api } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';
    flushMock.mockImplementationOnce(async () => {
      api.editingDraft.value = null;
    });
    (ElMessageBox.confirm as any).mockResolvedValueOnce(true);
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '2', Name: 'C' } })).toBe(true);
    expect(api.editingRowId.value).toBe('2');
  });
});
