// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { computed, defineComponent, h, nextTick, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';

import type { UseField } from '@/web/web/composables/useField';
import OManyToOneField from './OManyToOneField.vue';
import OManyToOneRefField from './OManyToOneRefField.vue';
import OManyToManyTagsField from './OManyToManyTagsField.vue';
import OManyToManyRefTagsField from './OManyToManyRefTagsField.vue';

vi.mock('@/auth/web/composables/usePermission', () => ({
  usePermission: () => ({
    hasAction: () => true,
  }),
}));

vi.mock('element-plus', async importOriginal => {
  const mod = await importOriginal<typeof import('element-plus')>();
  return {
    ...mod,
    ElMessage: { error: vi.fn(), warning: vi.fn(), success: vi.fn() },
  };
});

function makeM2OBinding(relationStore: any): UseField {
  const value = ref<any>(null);
  const record = ref({ Id: '1' });
  return {
    env: {
      isForm: true,
      isEditMode: true,
      viewMode: 'edit',
      fieldPrefix: null,
    },
    prop: 'PartnerId',
    meta: { type: 'ManyToOne', relationModel: 'demo.Partner' } as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => record.value) as any,
    registerFields: () => {},
    relationStore,
    store: undefined,
    asView: () => ({ fieldValue: () => value }) as any,
  } as UseField;
}

function makeM2MBinding(relationStore: any): UseField {
  const items = ref<any[]>([]);
  const record = ref({ Id: '1' });
  return {
    env: {
      isForm: true,
      isEditMode: true,
      viewMode: 'edit',
      fieldPrefix: null,
    },
    prop: 'TagIds',
    meta: { type: 'ManyToMany', relationModel: 'demo.Tag' } as any,
    fieldRef: () => items as any,
    fieldRefOf: () => items as any,
    recordRef: () => computed(() => record.value) as any,
    registerFields: () => {},
    relationStore,
    store: undefined,
    asMutableArray: () => ({
      getItems: () => items.value,
      insertItem: (row: any) => {
        items.value = [...items.value, row];
      },
      clearItems: () => {
        items.value = [];
      },
    }),
    asView: () => ({ fieldValue: () => items }) as any,
  } as UseField;
}

/** Stub that forwards remoteMethod with a chosen query payload. */
const SelectV2Stub = defineComponent({
  name: 'ElSelectV2',
  props: {
    remoteMethod: { type: Function, default: undefined },
  },
  setup(props, { slots }) {
    return () =>
      h('div', { class: 'select-stub' }, [
        h('button', {
          class: 'trigger-remote',
          type: 'button',
          onClick: () => props.remoteMethod?.('  alice  '),
        }),
        h('button', {
          class: 'trigger-remote-null',
          type: 'button',
          onClick: () => props.remoteMethod?.(null as any),
        }),
        h('button', {
          class: 'trigger-remote-empty',
          type: 'button',
          onClick: () => props.remoteMethod?.(''),
        }),
        slots.footer?.(),
      ]);
  },
});

const fieldStubs = {
  OFieldBase: defineComponent({
    props: {
      binding: { type: Object, required: true },
    },
    setup(props, { slots }) {
      const fieldValue = () => (props.binding as UseField).fieldRef();
      return () =>
        h('div', { class: 'ob' }, [
          slots.edit?.({ fieldValue, record: { Id: '1' } }),
          slots.display?.({ fieldValue, record: { Id: '1' } }),
        ]);
    },
  }),
  'el-select-v2': SelectV2Stub,
  ElSelectV2: SelectV2Stub,
  'el-dialog': true,
  ElDialog: true,
  'el-button': true,
  ElButton: true,
  'el-tag': true,
  ElTag: true,
  OViewScope: true,
};

async function clickRemote(wrapper: ReturnType<typeof mount>, cls = '.trigger-remote') {
  await wrapper.get(cls).trigger('click');
  await flushPromises();
  await nextTick();
}

describe('relation typeahead NameSearch wiring', () => {
  it('OManyToOneField remote search calls NameSearch with trimmed keyword', async () => {
    const NameSearch = vi.fn(async () => [{ Id: 'p1', DisplayName: 'Alice' }]);
    const Search = vi.fn();
    const wrapper = mount(OManyToOneField as any, {
      props: {
        binding: makeM2OBinding({ NameSearch, Search }),
        renderMode: 'form',
        pageSize: 15,
      },
      global: { stubs: fieldStubs },
    });

    await clickRemote(wrapper);

    expect(NameSearch).toHaveBeenCalledTimes(1);
    expect(NameSearch).toHaveBeenCalledWith('alice', [], {
      fields: ['Id', 'DisplayName'],
      limit: 15,
    });
    expect(Search).not.toHaveBeenCalled();
  });

  it('OManyToOneField covers nullish query and missing relationStore branches', async () => {
    const NameSearch = vi.fn(async () => undefined);
    const withStore = mount(OManyToOneField as any, {
      props: {
        binding: makeM2OBinding({ NameSearch }),
        renderMode: 'form',
      },
      global: { stubs: fieldStubs },
    });
    // Hit `query ?? ''` when remote-method passes null.
    await clickRemote(withStore, '.trigger-remote-null');
    expect(NameSearch).toHaveBeenCalledWith('', [], {
      fields: ['Id', 'DisplayName'],
      limit: 20,
    });

    // Hit `relationStore?.` short-circuit when store is unresolved.
    NameSearch.mockClear();
    const withoutStore = mount(OManyToOneField as any, {
      props: {
        binding: makeM2OBinding(undefined),
        renderMode: 'form',
      },
      global: { stubs: fieldStubs },
    });
    await clickRemote(withoutStore);
    expect(NameSearch).not.toHaveBeenCalled();
  });

  it('OManyToOneRefField remote search calls NameSearch with trimmed keyword', async () => {
    const NameSearch = vi.fn(async () => [{ Id: 'p1', DisplayName: 'Alice' }]);
    const Search = vi.fn();
    const wrapper = mount(OManyToOneRefField as any, {
      props: {
        binding: makeM2OBinding({ NameSearch, Search }),
        renderMode: 'form',
        pageSize: 12,
      },
      global: { stubs: fieldStubs },
    });

    await clickRemote(wrapper);

    expect(NameSearch).toHaveBeenCalledTimes(1);
    expect(NameSearch).toHaveBeenCalledWith('alice', [], {
      fields: ['Id', 'DisplayName'],
      limit: 12,
    });
    expect(Search).not.toHaveBeenCalled();
  });

  it('OManyToOneRefField covers nullish query and missing relationStore branches', async () => {
    const NameSearch = vi.fn(async () => undefined);
    const withStore = mount(OManyToOneRefField as any, {
      props: {
        binding: makeM2OBinding({ NameSearch }),
        renderMode: 'form',
      },
      global: { stubs: fieldStubs },
    });
    await clickRemote(withStore, '.trigger-remote-null');
    expect(NameSearch).toHaveBeenCalledWith('', [], {
      fields: ['Id', 'DisplayName'],
      limit: 20,
    });

    NameSearch.mockClear();
    const withoutStore = mount(OManyToOneRefField as any, {
      props: {
        binding: makeM2OBinding(undefined),
        renderMode: 'form',
      },
      global: { stubs: fieldStubs },
    });
    await clickRemote(withoutStore);
    expect(NameSearch).not.toHaveBeenCalled();
  });

  it('OManyToManyTagsField remote search calls NameSearch with effective conditions', async () => {
    const NameSearch = vi.fn(async () => [{ Id: 't1', DisplayName: 'Alice' }]);
    const Search = vi.fn();
    const wrapper = mount(OManyToManyTagsField as any, {
      props: {
        binding: makeM2MBinding({ NameSearch, Search }),
        renderMode: 'form',
        suggestLimit: 8,
      },
      global: { stubs: fieldStubs },
    });

    await clickRemote(wrapper);

    expect(NameSearch).toHaveBeenCalledTimes(1);
    const [keyword, condition, options] = NameSearch.mock.calls[0]!;
    expect(keyword).toBe('alice');
    expect(condition).toEqual([]);
    expect(options).toEqual(
      expect.objectContaining({
        limit: 8,
        fields: expect.arrayContaining(['Id', 'DisplayName']),
      })
    );
    expect(Search).not.toHaveBeenCalled();
  });

  it('OManyToManyTagsField covers falsy keyword branch for NameSearch', async () => {
    const NameSearch = vi.fn(async () => []);
    const wrapper = mount(OManyToManyTagsField as any, {
      props: {
        binding: makeM2MBinding({ NameSearch }),
        renderMode: 'form',
      },
      global: { stubs: fieldStubs },
    });

    await clickRemote(wrapper, '.trigger-remote-empty');
    expect(NameSearch).toHaveBeenCalledWith('', [], expect.objectContaining({ limit: 20 }));

    NameSearch.mockClear();
    await clickRemote(wrapper, '.trigger-remote-null');
    expect(NameSearch).toHaveBeenCalledWith('', [], expect.objectContaining({ limit: 20 }));
  });

  it('OManyToManyRefTagsField remote search calls NameSearch with hydration fields', async () => {
    const NameSearch = vi.fn(async () => [{ Id: 't1', DisplayName: 'Alice' }]);
    const Search = vi.fn(async () => []);
    const wrapper = mount(OManyToManyRefTagsField as any, {
      props: {
        binding: makeM2MBinding({ NameSearch, Search }),
        renderMode: 'form',
        suggestLimit: 9,
      },
      global: { stubs: fieldStubs },
    });

    await clickRemote(wrapper);

    expect(NameSearch).toHaveBeenCalledTimes(1);
    const [keyword, condition, options] = NameSearch.mock.calls[0]!;
    expect(keyword).toBe('alice');
    expect(condition).toEqual([]);
    expect(options).toEqual(
      expect.objectContaining({
        limit: 9,
        fields: expect.arrayContaining(['Id', 'DisplayName']),
      })
    );
    // Remote typeahead must not fall back to Search (Id-in hydrate is unused with empty selection).
    expect(Search).not.toHaveBeenCalled();
  });

  it('OManyToManyRefTagsField covers falsy keyword branch for NameSearch', async () => {
    const NameSearch = vi.fn(async () => []);
    const Search = vi.fn(async () => []);
    const wrapper = mount(OManyToManyRefTagsField as any, {
      props: {
        binding: makeM2MBinding({ NameSearch, Search }),
        renderMode: 'form',
      },
      global: { stubs: fieldStubs },
    });

    await clickRemote(wrapper, '.trigger-remote-empty');
    expect(NameSearch).toHaveBeenCalledWith('', [], expect.objectContaining({ limit: 20 }));
    expect(Search).not.toHaveBeenCalled();

    NameSearch.mockClear();
    Search.mockClear();
    await clickRemote(wrapper, '.trigger-remote-null');
    expect(NameSearch).toHaveBeenCalledWith('', [], expect.objectContaining({ limit: 20 }));
    expect(Search).not.toHaveBeenCalled();
  });
});

describe('relation typeahead NameCreate wiring', () => {
  async function openCreate(wrapper: ReturnType<typeof mount>, testId: string) {
    await clickRemote(wrapper);
    return wrapper.get(`[data-testid="${testId}"]`);
  }

  it('OManyToOneField Create entry calls NameCreate and sets value', async () => {
    const { ElMessage } = await import('element-plus');
    const NameSearch = vi.fn(async () => []);
    const created = { Id: 'new1', DisplayName: 'alice', Name: 'alice' };
    const NameCreate = vi.fn(async (name: string) => ({ Id: 'new1', DisplayName: name, Name: name }));
    const binding = makeM2OBinding({
      NameSearch,
      NameCreate,
      fullModelName: 'partner.Partner',
    });
    const wrapper = mount(OManyToOneField as any, {
      props: {
        binding,
        renderMode: 'form',
        allowCreate: true,
        createActionId: '',
        nameField: 'Code',
      },
      global: { stubs: fieldStubs },
    });

    const createBtn = await openCreate(wrapper, 'o-m2o-name-create');
    expect(createBtn.text()).toContain('alice');
    await createBtn.trigger('click');
    await flushPromises();
    expect(NameCreate).toHaveBeenCalledWith('alice', undefined, { nameField: 'Code' });
    expect(binding.fieldRef().value).toEqual(created);

    NameCreate.mockImplementationOnce(async () => {
      throw new Error('denied');
    });
    await clickRemote(wrapper);
    const again = wrapper.get('[data-testid="o-m2o-name-create"]');
    await again.trigger('keydown.enter');
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('denied');
    NameCreate.mockImplementationOnce(async () => {
      throw new Error('denied2');
    });
    await clickRemote(wrapper);
    await wrapper.get('[data-testid="o-m2o-name-create"]').trigger('keydown.space');
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('denied2');
  });

  it('hides Create entry by default and when allowCreate is false', async () => {
    for (const [Comp, testId, makeBinding] of [
      [OManyToOneField, 'o-m2o-name-create', makeM2OBinding],
      [OManyToOneRefField, 'o-m2o-name-create', makeM2OBinding],
      [OManyToManyTagsField, 'o-m2m-name-create', makeM2MBinding],
      [OManyToManyRefTagsField, 'o-m2m-name-create', makeM2MBinding],
    ] as const) {
      for (const allowCreate of [undefined, false] as const) {
        const wrapper = mount(Comp as any, {
          props: {
            binding: makeBinding({
              NameSearch: vi.fn(async () => []),
              NameCreate: vi.fn(),
              Search: vi.fn(async () => []),
              fullModelName: 'partner.Partner',
            }),
            renderMode: 'form',
            ...(allowCreate === false ? { allowCreate: false } : {}),
            createActionId: '',
          },
          global: { stubs: fieldStubs },
        });
        await clickRemote(wrapper);
        expect(wrapper.find(`[data-testid="${testId}"]`).exists()).toBe(false);
      }
    }
  });

  it('OManyToOneRefField Create entry selects created row', async () => {
    const { ElMessage } = await import('element-plus');
    const NameCreate = vi.fn(async (name: string) => ({ Id: 'r1', DisplayName: name, Name: name }));
    const binding = makeM2OBinding({
      NameSearch: vi.fn(async () => []),
      NameCreate,
      fullModelName: 'partner.Partner',
    });
    const wrapper = mount(OManyToOneRefField as any, {
      props: { binding, renderMode: 'form', allowCreate: true, createActionId: '' },
      global: { stubs: fieldStubs },
    });
    const btn = await openCreate(wrapper, 'o-m2o-name-create');
    await btn.trigger('click');
    await flushPromises();
    expect(NameCreate).toHaveBeenCalledWith('alice', undefined, undefined);
    expect(binding.fieldRef().value).toEqual({ Id: 'r1', DisplayName: 'alice', Name: 'alice' });

    NameCreate.mockRejectedValueOnce(new Error('ref-denied'));
    await clickRemote(wrapper);
    const again = wrapper.get('[data-testid="o-m2o-name-create"]');
    await again.trigger('keydown.enter');
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('ref-denied');
    NameCreate.mockRejectedValueOnce(new Error('ref-denied2'));
    await clickRemote(wrapper);
    await wrapper.get('[data-testid="o-m2o-name-create"]').trigger('keydown.space');
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('ref-denied2');
  });

  it('OManyToManyTagsField Create entry appends tag id', async () => {
    const { ElMessage } = await import('element-plus');
    const NameCreate = vi.fn(async (name: string) => ({ Id: 't9', DisplayName: name, Name: name }));
    const binding = makeM2MBinding({
      NameSearch: vi.fn(async () => []),
      NameCreate,
      fullModelName: 'partner.Partner',
    });
    const wrapper = mount(OManyToManyTagsField as any, {
      props: { binding, renderMode: 'form', allowCreate: true, createActionId: '' },
      global: { stubs: fieldStubs },
    });
    const btn = await openCreate(wrapper, 'o-m2m-name-create');
    await btn.trigger('click');
    await flushPromises();
    expect(NameCreate).toHaveBeenCalledWith('alice', undefined, undefined);
    expect(binding.fieldRef().value.map((r: any) => r.Id ?? r)).toEqual(['t9']);

    // Already-selected id path: create again with same id should not duplicate.
    NameCreate.mockResolvedValueOnce({ Id: 't9', DisplayName: 'alice', Name: 'alice' });
    await clickRemote(wrapper);
    await wrapper.get('[data-testid="o-m2m-name-create"]').trigger('keydown.enter');
    await flushPromises();
    expect(binding.fieldRef().value.map((r: any) => r.Id ?? r)).toEqual(['t9']);

    NameCreate.mockRejectedValueOnce(new Error('m2m-denied'));
    await clickRemote(wrapper);
    await wrapper.get('[data-testid="o-m2m-name-create"]').trigger('keydown.space');
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('m2m-denied');
  });

  it('OManyToManyRefTagsField Create entry appends tag id', async () => {
    const { ElMessage } = await import('element-plus');
    const NameCreate = vi.fn(async (name: string) => ({ Id: 'rt1', DisplayName: name, Name: name }));
    const binding = makeM2MBinding({
      NameSearch: vi.fn(async () => []),
      NameCreate,
      Search: vi.fn(async () => []),
      fullModelName: 'partner.Partner',
    });
    const wrapper = mount(OManyToManyRefTagsField as any, {
      props: { binding, renderMode: 'form', allowCreate: true, createActionId: '', nameField: 'Title' },
      global: { stubs: fieldStubs },
    });
    const btn = await openCreate(wrapper, 'o-m2m-name-create');
    await btn.trigger('click');
    await flushPromises();
    expect(NameCreate).toHaveBeenCalledWith('alice', undefined, { nameField: 'Title' });
    expect(binding.fieldRef().value.map((r: any) => r.Id ?? r)).toEqual(['rt1']);

    // Already-selected id path.
    NameCreate.mockResolvedValueOnce({ Id: 'rt1', DisplayName: 'alice', Name: 'alice' });
    await clickRemote(wrapper);
    await wrapper.get('[data-testid="o-m2m-name-create"]').trigger('click');
    await flushPromises();
    expect(binding.fieldRef().value.map((r: any) => r.Id ?? r)).toEqual(['rt1']);

    NameCreate.mockRejectedValueOnce(new Error('m2m-ref-denied'));
    await clickRemote(wrapper);
    const again = wrapper.get('[data-testid="o-m2m-name-create"]');
    await again.trigger('keydown.enter');
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('m2m-ref-denied');
    NameCreate.mockRejectedValueOnce(new Error('m2m-ref-denied2'));
    await clickRemote(wrapper);
    await wrapper.get('[data-testid="o-m2m-name-create"]').trigger('keydown.space');
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('m2m-ref-denied2');
  });
});
