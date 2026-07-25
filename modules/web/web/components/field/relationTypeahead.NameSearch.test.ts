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
  setup(props) {
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
      ]);
  },
});

const fieldStubs = {
  OFieldBase: {
    props: ['binding'],
    template: `<div class="ob">
      <slot name="edit" :fieldValue="() => ({ value: null })" :record="{ Id: '1' }" />
      <slot name="display" :fieldValue="() => ({ value: null })" :record="{ Id: '1' }" />
    </div>`,
  },
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
