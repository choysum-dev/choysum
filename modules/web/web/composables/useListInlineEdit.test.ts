// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, inject, provide, ref } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';

import {
  disposeOnchange,
  provideOnchange,
} from '@/web/web/composables/useOnchange';
import { useListInlineEdit, type UseListInlineEditDeps } from '@/web/web/composables/useListInlineEdit';
import { fnRecorder, flushPromises, mountApp } from '@/web/web/__tests__/mountApp';

const origConfirm = ElMessageBox.confirm;
const origSuccess = ElMessage.success;
const origError = ElMessage.error;

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

function mountInline(opts?: {
  enabled?: boolean;
  onSaved?: () => void | Promise<void>;
  updateImpl?: (id: string, payload: any) => Promise<any>;
  deps?: UseListInlineEditDeps;
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
        deps: opts?.deps,
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
    expect(await api.save()).toBe(true);
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
    expect(await api.save()).toBe(true);
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
    await expectRejects(() => api.save(), 'boom');
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
    await expectRejects(() => api.enterEdit({ kind: 'record', payload: { Id: '2', Name: 'C' } }), 'fail');
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
    expect(await first).toBe(true);
    expect(UpdateById.calls.length).toBe(1);
    unmount();
  });

  test('dirty switch aborts when save returns false', async () => {
    const { api, unmount } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';
    confirm.mockImplementation(async () => {
      // Clear draft so save() returns false without throwing.
      api.editingDraft.value = null;
      return true;
    });
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '2', Name: 'C' } })).toBe(false);
    expect(api.editingRowId.value).toBe('1');
    unmount();
  });

  test('exitEdit tolerates a missing onchange controller', async () => {
    const { api, unmount } = mountInline({
      deps: {
        useProvidedOnchange: () => null,
      },
    });
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    await api.discard();
    expect(api.isEditing.value).toBe(false);
    unmount();
  });

  test('provideOnchange uses draft root while editing and store fallback when idle', async () => {
    let capturedOpts: any;
    const { api, enabled, store, unmount } = mountInline({
      deps: {
        provideOnchange: ((s, session, opts) => {
          capturedOpts = opts;
          return provideOnchange(s, session, opts);
        }) as any,
      },
    });

    expect(capturedOpts.getRoot().Id).toBe('store-rec');

    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    expect(capturedOpts.getRoot().Id).toBe('1');
    capturedOpts.onPatch({ Name: 'Z' });
    expect(api.editingDraft.value?.Name).toBe('Z');
    // Truthy non-object must hit the typeof!=='object' branch while a draft exists.
    capturedOpts.onPatch('x');
    capturedOpts.onPatch(42);
    expect(api.editingDraft.value?.Name).toBe('Z');

    api.editingDraft.value = null;
    expect(capturedOpts.getRoot().Id).toBe('store-rec');
    capturedOpts.onPatch({ Name: 'ignored' });
    capturedOpts.onPatch(null);
    capturedOpts.onPatch('x');
    expect(api.editingDraft.value).toBeNull();

    enabled.value = false;
    expect(capturedOpts.getRoot().Id).toBe('store-rec');
    store.state._draftRecord = { Id: 'draft', Name: 'D' };
    expect(capturedOpts.getRoot().Id).toBe('draft');
    unmount();
  });

  test('re-pauses onchange after reset on enter and exit edit', async () => {
    const { api, Onchange, unmount } = mountInline();
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    Onchange.mockClear();
    // Controller is paused after enter — draft mutation must not auto-RPC.
    api.editingDraft.value = { Id: '1', Name: 'B' };
    await flushPromises();
    await sleep(40);
    expect(Onchange.calls.length).toBe(0);

    await api.discard();
    Onchange.mockClear();
    // After discard, controller is reset+paused again; mutation must not auto-RPC.
    api.editingDraft.value = { Id: '1', Name: 'C' };
    await flushPromises();
    await sleep(40);
    expect(Onchange.calls.length).toBe(0);
    unmount();
  });

  test('exits edit when flush clears draft during save', async () => {
    const afterFlushHandlers = new Set<(p: any) => void>();
    const flush = fnRecorder(async () => undefined);
    const oc = {
      flush,
      reset: fnRecorder(),
      pause: fnRecorder(),
      force: fnRecorder(),
      running: ref(false),
      registerAfterFlush: (cb: (p: any) => void) => {
        afterFlushHandlers.add(cb);
      },
      unregisterAfterFlush: (cb: (p: any) => void) => {
        afterFlushHandlers.delete(cb);
      },
    };
    const { api, UpdateById, unmount } = mountInline({
      deps: {
        provideOnchange: (() => oc) as any,
        useProvidedOnchange: (() => oc) as any,
      },
    });
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';
    flush.mockImplementation(async () => {
      api.editingDraft.value = null;
    });
    expect(await api.save()).toBe(true);
    expect(UpdateById.calls.length).toBe(0);
    expect(api.isEditing.value).toBe(false);
    expect(api.editingRowId.value).toBeNull();
    unmount();
  });

  test('blocks UpdateById when onchange flush reports error messages', async () => {
    const afterFlushHandlers = new Set<(p: any) => void>();
    let flushImpl: () => Promise<void> = async () => undefined;
    const oc = {
      flush: async () => flushImpl(),
      reset: fnRecorder(),
      pause: fnRecorder(),
      force: fnRecorder(),
      running: ref(false),
      registerAfterFlush: (cb: (p: any) => void) => {
        afterFlushHandlers.add(cb);
      },
      unregisterAfterFlush: (cb: (p: any) => void) => {
        afterFlushHandlers.delete(cb);
      },
    };
    const { api, UpdateById, unmount } = mountInline({
      deps: {
        provideOnchange: (() => oc) as any,
        useProvidedOnchange: (() => oc) as any,
      },
    });
    flushImpl = async () => {
      for (const cb of afterFlushHandlers) {
        cb({ result: { messages: [{ level: 'error', message: 'bad' }] } });
      }
    };
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';
    expect(await api.save()).toBe(false);
    expect(UpdateById.calls.length).toBe(0);
    expect(api.isEditing.value).toBe(true);
    expect(error.calls.length).toBeGreaterThan(0);
    unmount();
  });

  test('dirty switch continues after flush-cleared draft save exits edit', async () => {
    const afterFlushHandlers = new Set<(p: any) => void>();
    const flush = fnRecorder(async () => undefined);
    const oc = {
      flush,
      reset: fnRecorder(),
      pause: fnRecorder(),
      force: fnRecorder(),
      running: ref(false),
      registerAfterFlush: (cb: (p: any) => void) => {
        afterFlushHandlers.add(cb);
      },
      unregisterAfterFlush: (cb: (p: any) => void) => {
        afterFlushHandlers.delete(cb);
      },
    };
    const { api, UpdateById, unmount } = mountInline({
      deps: {
        provideOnchange: (() => oc) as any,
        useProvidedOnchange: (() => oc) as any,
      },
    });
    await api.enterEdit({ kind: 'record', payload: { Id: '1', Name: 'A' } });
    api.editingDraft.value!.Name = 'B';
    flush.mockImplementation(async () => {
      api.editingDraft.value = null;
    });
    confirm.mockImplementation(async () => true);
    expect(await api.enterEdit({ kind: 'record', payload: { Id: '2', Name: 'C' } })).toBe(true);
    expect(UpdateById.calls.length).toBe(0);
    expect(api.editingRowId.value).toBe('2');
    unmount();
  });
});
