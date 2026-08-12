// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { defineComponent, h, nextTick } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import OPropertiesDefinitionEditor from './OPropertiesDefinitionEditor.vue';
import {
  buildDefinitionScopeCondition,
  definitionItemsToDrafts,
  draftsToDefinitionItems,
  emptyDraftItem,
} from './oproperties_definition_helpers';

const { createStoreByModel, draftsToDefinitionItemsMock, realDraftsToDefinitionItems } = vi.hoisted(() => ({
  createStoreByModel: vi.fn(),
  draftsToDefinitionItemsMock: vi.fn(),
  realDraftsToDefinitionItems: { current: null as null | ((...args: any[]) => unknown) },
}));

vi.mock('./oproperties_definition_helpers', async importOriginal => {
  const actual = await importOriginal<typeof import('./oproperties_definition_helpers')>();
  realDraftsToDefinitionItems.current = actual.draftsToDefinitionItems as any;
  draftsToDefinitionItemsMock.mockImplementation((...args: any[]) =>
    (realDraftsToDefinitionItems.current as any)(...args)
  );
  return {
    ...actual,
    draftsToDefinitionItems: draftsToDefinitionItemsMock,
  };
});

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({
      _t: (msg: string) => msg,
    }),
  };
});

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel,
}));

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  const Stub = (name: string, tag = 'input') =>
    defineComponent({
      name,
      props: {
        modelValue: { type: [String, Number, Boolean, Object], default: undefined },
        disabled: Boolean,
        type: String,
        size: String,
        placeholder: String,
        loading: Boolean,
        plain: Boolean,
        autosize: [Boolean, Object],
      },
      emits: ['update:modelValue', 'click'],
      setup(props, { emit, slots, attrs }) {
        if (name === 'ElButton') {
          return () =>
            h(
              'button',
              {
                ...attrs,
                disabled: props.disabled || undefined,
                onClick: (e: Event) => {
                  emit('click', e);
                },
              },
              slots.default?.()
            );
        }
        if (name === 'ElSwitch') {
          return () =>
            h('input', {
              ...attrs,
              type: 'checkbox',
              checked: !!props.modelValue,
              disabled: props.disabled || undefined,
              onChange: (e: Event) => emit('update:modelValue', (e.target as HTMLInputElement).checked),
            });
        }
        if (name === 'ElSelect') {
          return () =>
            h(
              'select',
              {
                ...attrs,
                value: props.modelValue == null ? '' : String(props.modelValue),
                disabled: props.disabled || undefined,
                onChange: (e: Event) => emit('update:modelValue', (e.target as HTMLSelectElement).value),
              },
              slots.default?.()
            );
        }
        if (name === 'ElOption') {
          return () => h('option', { value: (attrs as any).value }, (attrs as any).label || slots.default?.());
        }
        return () =>
          h(tag, {
            ...attrs,
            value: props.modelValue == null ? '' : String(props.modelValue),
            disabled: props.disabled || undefined,
            onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLInputElement).value),
          });
      },
    });
  return {
    ...actual,
    ElButton: Stub('ElButton', 'button'),
    ElInput: Stub('ElInput', 'textarea'),
    ElSelect: Stub('ElSelect'),
    ElOption: Stub('ElOption'),
    ElSwitch: Stub('ElSwitch'),
  };
});

describe('oproperties_definition_helpers', () => {
  it('covers drafts, coerce defaults, selection errors, and scope conditions', () => {
    expect(emptyDraftItem().type).toBe('char');
    expect(definitionItemsToDrafts(null)).toEqual([]);
    expect(definitionItemsToDrafts('x' as any)).toEqual([]);
    expect(definitionItemsToDrafts([null, 's', [], { name: '' }, { name: 'ok', type: 'char' }])).toEqual([
      expect.objectContaining({ name: 'ok', type: 'char' }),
    ]);
    // falsy/missing type → default 'char'
    expect(definitionItemsToDrafts([{ name: 'notyped' }])[0]?.type).toBe('char');
    expect(definitionItemsToDrafts([{ name: 'emptytype', type: '' }])[0]?.type).toBe('char');

    const circ: any[] = [];
    circ.push(circ);
    expect(definitionItemsToDrafts([{ name: 'c', type: 'selection', selection: circ }])[0]?.selectionText).toBe('');

    const drafts = definitionItemsToDrafts([
      { name: 'code', type: 'char', string: 'Code', default: 'A', readonly: true },
      { name: 'kind', type: 'selection', selection: [['a', 'A']] },
      { name: 'weird', type: 'not-a-real-type' },
      { name: 'emptyDefaults', type: 'char', string: null, default: null },
      { name: '', type: 'char' },
    ]);
    expect(drafts).toHaveLength(4);
    expect(drafts[1]?.selectionText).toContain('a');
    expect(drafts[2]?.type).toBe('not-a-real-type');

    expect(draftsToDefinitionItems(null)).toEqual([]);
    expect(draftsToDefinitionItems(undefined)).toEqual([]);
    expect(
      draftsToDefinitionItems([
        { ...emptyDraftItem(), name: 'b', type: 'boolean', default: 'true' },
        { ...emptyDraftItem(), name: 'b0', type: 'boolean', default: '0' },
        { ...emptyDraftItem(), name: 'b1', type: 'boolean', default: '1' },
        { ...emptyDraftItem(), name: 'bf', type: 'boolean', default: 'false' },
        { ...emptyDraftItem(), name: 'bx', type: 'boolean', default: 'maybe' },
        { ...emptyDraftItem(), name: 'i', type: 'integer', default: '3.9' },
        { ...emptyDraftItem(), name: 'ibad', type: 'integer', default: 'nope' },
        { ...emptyDraftItem(), name: 'f', type: 'float', default: '1.5' },
        { ...emptyDraftItem(), name: 'fbad', type: 'float', default: 'nope' },
        { ...emptyDraftItem(), name: 'c', type: 'char', default: 'x', string: 'L', readonly: true },
        {
          ...emptyDraftItem(),
          name: 'sel',
          type: 'selection',
          selectionText: '[["a","A"]]',
        },
        { ...emptyDraftItem(), name: '', type: 'char' },
        // nullish default uses ?? '' (no default emitted)
        { ...emptyDraftItem(), name: 'ndef', type: 'char', default: null as any },
        { ...emptyDraftItem(), name: 'udef', type: 'char', default: undefined as any },
      ])
    ).toEqual([
      { name: 'b', type: 'boolean', default: true },
      { name: 'b0', type: 'boolean', default: false },
      { name: 'b1', type: 'boolean', default: true },
      { name: 'bf', type: 'boolean', default: false },
      { name: 'bx', type: 'boolean', default: 'maybe' },
      { name: 'i', type: 'integer', default: 3 },
      { name: 'ibad', type: 'integer', default: 'nope' },
      { name: 'f', type: 'float', default: 1.5 },
      { name: 'fbad', type: 'float', default: 'nope' },
      { name: 'c', type: 'char', string: 'L', default: 'x', readonly: true },
      { name: 'sel', type: 'selection', selection: [['a', 'A']] },
      { name: 'ndef', type: 'char' },
      { name: 'udef', type: 'char' },
    ]);

    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 'x', type: 'html' }])
    ).toThrow(/unsupported property type/);
    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '' }])
    ).toThrow(/requires selection JSON/);
    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '{' }])
    ).toThrow(/invalid JSON/);
    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '{}' }])
    ).toThrow(/non-empty selection array/);
    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '[]' }])
    ).toThrow(/non-empty selection array/);

    expect(
      buildDefinitionScopeCondition({
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
      })
    ).toEqual([
      ['TargetModel', '=', 'Partner'],
      ['PropertiesField', '=', 'PartnerProperties'],
      ['ContainerModel', '=', null],
      ['ContainerId', '=', null],
    ]);
    expect(
      buildDefinitionScopeCondition({
        targetModel: 'Task',
        propertiesField: 'TaskProperties',
        containerModel: 'Project',
        containerId: 'p1',
      })
    ).toEqual([
      ['TargetModel', '=', 'Task'],
      ['PropertiesField', '=', 'TaskProperties'],
      ['ContainerModel', '=', 'Project'],
      ['ContainerId', '=', 'p1'],
    ]);
    expect(
      buildDefinitionScopeCondition({
        targetModel: 'Task',
        propertiesField: 'TaskProperties',
        containerModel: '',
        containerId: '',
      })
    ).toEqual([
      ['TargetModel', '=', 'Task'],
      ['PropertiesField', '=', 'TaskProperties'],
      ['ContainerModel', '=', null],
      ['ContainerId', '=', null],
    ]);
    // falsy model/field names still emit trimmed empty strings in the condition
    expect(
      buildDefinitionScopeCondition({
        targetModel: '',
        propertiesField: null as any,
        containerModel: null,
        containerId: null,
      })
    ).toEqual([
      ['TargetModel', '=', ''],
      ['PropertiesField', '=', ''],
      ['ContainerModel', '=', null],
      ['ContainerId', '=', null],
    ]);
  });
});

describe('OPropertiesDefinitionEditor', () => {
  beforeEach(() => {
    createStoreByModel.mockReset();
    createStoreByModel.mockImplementation(() => {
      throw new Error('createStoreByModel not stubbed');
    });
    draftsToDefinitionItemsMock.mockReset();
    draftsToDefinitionItemsMock.mockImplementation((...args: any[]) =>
      (realDraftsToDefinitionItems.current as any)(...args)
    );
  });

  it('loads App-level definition and saves UpdateById', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const Search = vi.fn(async () => [
      {
        Id: 'def-1',
        Definition: [{ name: 'tax_id', type: 'char', string: 'Tax' }],
      },
    ]);
    const UpdateById = vi.fn(async () => ({ Id: 'def-1' }));
    const Create = vi.fn();
    const store = { Search, UpdateById, Create } as any;

    const wrapper = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        store,
      },
    });
    try {
      await flushPromises();
      expect(Search).toHaveBeenCalledWith(
        {
          And: [
            ['TargetModel', '=', 'Partner'],
            ['PropertiesField', '=', 'PartnerProperties'],
            ['ContainerModel', '=', null],
            ['ContainerId', '=', null],
          ],
        },
        expect.any(Object)
      );
      expect(wrapper.find('[data-testid="o-properties-definition-name"]').exists()).toBe(true);

      await wrapper.find('[data-testid="o-properties-definition-save"]').trigger('click');
      await flushPromises();
      expect(UpdateById).toHaveBeenCalledWith(
        'def-1',
        expect.objectContaining({
          Definition: [expect.objectContaining({ name: 'tax_id', type: 'char' })],
        })
      );
      expect(Create).not.toHaveBeenCalled();
      expect(warn).not.toHaveBeenCalled();
    } finally {
      wrapper.unmount();
      warn.mockRestore();
    }
  });

  it('creates parent-container definition when none exists', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const Search = vi.fn(async () => []);
    const Create = vi.fn(async (vals: any) => ({ Id: 'new-1', ...vals }));
    const UpdateById = vi.fn();
    const store = { Search, Create, UpdateById } as any;

    const wrapper = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'project',
        targetModel: 'Task',
        propertiesField: 'TaskProperties',
        containerModel: 'Project',
        containerId: 'proj-9',
        store,
      },
    });
    try {
      await flushPromises();
      expect(wrapper.find('[data-testid="o-properties-definition-empty"]').exists()).toBe(true);

      await wrapper.find('[data-testid="o-properties-definition-add"]').trigger('click');
      await nextTick();
      const nameInput = wrapper.find('[data-testid="o-properties-definition-name"]');
      await nameInput.setValue('prio');
      await wrapper.find('[data-testid="o-properties-definition-save"]').trigger('click');
      await flushPromises();

      expect(Create).toHaveBeenCalledWith(
        expect.objectContaining({
          TargetModel: 'Task',
          PropertiesField: 'TaskProperties',
          ContainerModel: 'Project',
          ContainerId: 'proj-9',
          Definition: [expect.objectContaining({ name: 'prio', type: 'char' })],
        })
      );
      expect(UpdateById).not.toHaveBeenCalled();
      expect(warn).not.toHaveBeenCalled();
    } finally {
      wrapper.unmount();
      warn.mockRestore();
    }
  });

  it('surfaces save errors without leaking unhandled warns when spied', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const Search = vi.fn(async () => []);
    const Create = vi.fn(async () => {
      throw new Error('no write acl');
    });
    const store = { Search, Create, UpdateById: vi.fn() } as any;
    const wrapper = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        store,
      },
    });
    try {
      await flushPromises();
      await wrapper.find('[data-testid="o-properties-definition-add"]').trigger('click');
      await nextTick();
      await wrapper.find('[data-testid="o-properties-definition-name"]').setValue('x');
      await wrapper.find('[data-testid="o-properties-definition-save"]').trigger('click');
      await flushPromises();
      expect(wrapper.find('[data-testid="o-properties-definition-save-error"]').text()).toContain('no write acl');
      expect(warn).toHaveBeenCalledWith('[OPropertiesDefinitionEditor] save failed', expect.any(Error));
    } finally {
      wrapper.unmount();
      warn.mockRestore();
    }
  });

  it('covers store resolution, load errors, readonly, remove, and stale reload', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});

    const emptyApp = mount(OPropertiesDefinitionEditor as any, {
      props: { application: '  ', targetModel: 'T', propertiesField: 'F' },
    });
    await flushPromises();
    expect(emptyApp.find('[data-testid="o-properties-definition-save"]').attributes('disabled')).toBeDefined();
    emptyApp.unmount();

    // canSave || '' fallbacks for missing/empty required props (avoid && short-circuit)
    const missingFields = mount(OPropertiesDefinitionEditor as any, {
      props: { application: 'partner', targetModel: 'Partner', propertiesField: '' },
    });
    await flushPromises();
    expect(missingFields.find('[data-testid="o-properties-definition-save"]').attributes('disabled')).toBeDefined();
    missingFields.unmount();

    const undefField = mount(OPropertiesDefinitionEditor as any, {
      props: { application: 'partner', targetModel: 'Partner', propertiesField: undefined as any },
    });
    await flushPromises();
    expect(undefField.find('[data-testid="o-properties-definition-save"]').attributes('disabled')).toBeDefined();
    undefField.unmount();

    // resolveStore without application (setupState — canSave guards normally skip this)
    const noApp = mount(OPropertiesDefinitionEditor as any, {
      props: { application: '', targetModel: 'T', propertiesField: 'F', store: null },
    });
    await flushPromises();
    expect((noApp.vm as any).$.setupState.resolveStore()).toBeNull();
    noApp.unmount();

    createStoreByModel.mockImplementation(() => {
      throw new Error('no store registry');
    });
    const badRegistry = mount(OPropertiesDefinitionEditor as any, {
      props: { application: 'partner', targetModel: 'Partner', propertiesField: 'PartnerProperties' },
    });
    await flushPromises();
    expect(badRegistry.find('[data-testid="o-properties-definition-error"]').text()).toContain('no store registry');
    badRegistry.unmount();

    createStoreByModel.mockReturnValue({});
    const noSearch = mount(OPropertiesDefinitionEditor as any, {
      props: { application: 'partner', targetModel: 'Partner', propertiesField: 'PartnerProperties' },
    });
    await flushPromises();
    expect(noSearch.find('[data-testid="o-properties-definition-error"]').text()).toContain(
      'PropertyDefinition store is unavailable'
    );
    noSearch.unmount();

    const resolvers: Array<(v: any) => void> = [];
    const Search = vi.fn(
      () =>
        new Promise(resolve => {
          resolvers.push(resolve);
        })
    );
    const store = { Search, Create: vi.fn(), UpdateById: vi.fn() } as any;
    const stale = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        store,
      },
    });
    await nextTick();
    expect(resolvers.length).toBe(1);
    await stale.setProps({ containerId: 'c2', containerModel: 'C' });
    await nextTick();
    expect(resolvers.length).toBe(2);
    resolvers[0]!([{ Id: 'stale', Definition: [{ name: 'old', type: 'char' }] }]);
    await flushPromises();
    // Stale first response ignored while second reload still pending.
    expect(stale.find('[data-testid="o-properties-definition-name"]').exists()).toBe(false);
    resolvers[1]!([]);
    await flushPromises();
    stale.unmount();

    // Stale reject path (seq !== reloadSeq in catch) + Error load message
    const staleRejecters: Array<(e: any) => void> = [];
    const staleResolvers: Array<(v: any) => void> = [];
    const SearchStaleFail = vi.fn(
      () =>
        new Promise((resolve, reject) => {
          staleResolvers.push(resolve);
          staleRejecters.push(reject);
        })
    );
    const staleFail = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        store: { Search: SearchStaleFail, Create: vi.fn(), UpdateById: vi.fn() },
      },
    });
    await nextTick();
    await staleFail.setProps({ containerId: 'c3' });
    await nextTick();
    expect(staleRejecters.length).toBe(2);
    staleRejecters[0]!(new Error('stale-load-error'));
    await flushPromises();
    expect(staleFail.find('[data-testid="o-properties-definition-error"]').exists()).toBe(false);
    staleResolvers[1]!([]);
    await flushPromises();
    staleFail.unmount();

    const loadFailSearch = vi.fn(async () => {
      throw 'bare-load-fail';
    });
    const loadFail = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        store: { Search: loadFailSearch, Create: vi.fn(), UpdateById: vi.fn() },
      },
    });
    await flushPromises();
    expect(loadFail.find('[data-testid="o-properties-definition-error"]').text()).toContain('bare-load-fail');
    expect(warn).toHaveBeenCalledWith('[OPropertiesDefinitionEditor] load failed', 'bare-load-fail');
    loadFail.unmount();

    const loadFailErrSearch = vi.fn(async () => {
      throw new Error('typed-load-fail');
    });
    const loadFailErr = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        store: { Search: loadFailErrSearch, Create: vi.fn(), UpdateById: vi.fn() },
      },
    });
    await flushPromises();
    expect(loadFailErr.find('[data-testid="o-properties-definition-error"]').text()).toContain('typed-load-fail');
    loadFailErr.unmount();

    const Search2 = vi.fn(async () => []);
    const Create2 = vi.fn(async (vals: any) => ({ Id: 'created', ...vals }));
    const wrapper = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        store: { Search: Search2, Create: Create2, UpdateById: vi.fn() },
      },
    });
    await flushPromises();
    await wrapper.find('[data-testid="o-properties-definition-add"]').trigger('click');
    await nextTick();
    await wrapper.find('[data-testid="o-properties-definition-name"]').setValue('n1');
    await wrapper.find('[data-testid="o-properties-definition-string"]').setValue('Label');
    await wrapper.find('[data-testid="o-properties-definition-default"]').setValue('D');
    await wrapper.find('[data-testid="o-properties-definition-readonly"]').setValue(true);
    await wrapper.find('[data-testid="o-properties-definition-type"]').setValue('selection');
    await nextTick();
    expect(wrapper.find('[data-testid="o-properties-definition-selection"]').exists()).toBe(true);
    await wrapper.find('[data-testid="o-properties-definition-selection"]').setValue('[["a","A"]]');
    await wrapper.find('[data-testid="o-properties-definition-remove"]').trigger('click');
    await nextTick();
    expect(wrapper.find('[data-testid="o-properties-definition-empty"]').exists()).toBe(true);

    await wrapper.find('[data-testid="o-properties-definition-add"]').trigger('click');
    await nextTick();
    await wrapper.find('[data-testid="o-properties-definition-name"]').setValue('bad');
    await wrapper.find('[data-testid="o-properties-definition-type"]').setValue('html');
    await wrapper.find('[data-testid="o-properties-definition-save"]').trigger('click');
    await flushPromises();
    expect(wrapper.find('[data-testid="o-properties-definition-save-error"]').text()).toMatch(
      /unsupported property type/
    );

    // non-Error throw from draftsToDefinitionItems → String(e) saveError path
    draftsToDefinitionItemsMock.mockImplementationOnce(() => {
      throw 'bare-draft-fail';
    });
    await wrapper.find('[data-testid="o-properties-definition-type"]').setValue('char');
    await wrapper.find('[data-testid="o-properties-definition-save"]').trigger('click');
    await flushPromises();
    expect(wrapper.find('[data-testid="o-properties-definition-save-error"]').text()).toContain(
      'bare-draft-fail'
    );
    wrapper.unmount();

    // null/missing Id → definitionId null via Id || ''; blank container dims → null payload
    const CreateNoId = vi.fn(async () => ({}));
    const createWrapper = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        containerModel: '',
        containerId: '',
        store: {
          Search: vi.fn(async () => [{ Id: null, Definition: [{ name: 'a', type: 'char' }] }]),
          Create: CreateNoId,
          UpdateById: vi.fn(),
        },
      },
    });
    await flushPromises();
    expect(createWrapper.find('[data-testid="o-properties-definition-name"]').exists()).toBe(true);
    await createWrapper.find('[data-testid="o-properties-definition-save"]').trigger('click');
    await flushPromises();
    expect(CreateNoId).toHaveBeenCalledWith(
      expect.objectContaining({ ContainerModel: null, ContainerId: null })
    );
    createWrapper.unmount();

    const whitespaceId = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        store: {
          Search: vi.fn(async () => [{ Id: '  ', Definition: [{ name: 'a', type: 'char' }] }]),
          Create: vi.fn(async () => ({ Id: 'x' })),
          UpdateById: vi.fn(),
        },
      },
    });
    await flushPromises();
    whitespaceId.unmount();

    const readonly = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        readonly: true,
        store: {
          Search: vi.fn(async () => [{ Id: 'r1', Definition: [{ name: 'a', type: 'char' }] }]),
          Create: vi.fn(),
          UpdateById: vi.fn(),
        },
      },
    });
    await flushPromises();
    // disabled buttons may not fire click — call guards via setupState
    (readonly.vm as any).$.setupState.onAdd();
    await (readonly.vm as any).$.setupState.onSave();
    await flushPromises();
    expect(readonly.find('[data-testid="o-properties-definition-name"]').exists()).toBe(true);
    readonly.unmount();

    createStoreByModel.mockImplementation(() => {
      throw 'registry-string-fail';
    });
    const noStoreSave = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        store: null,
      },
    });
    await flushPromises();
    expect(noStoreSave.find('[data-testid="o-properties-definition-error"]').text()).toContain(
      'registry-string-fail'
    );
    noStoreSave.unmount();

    createStoreByModel.mockReset();
    createStoreByModel.mockReturnValue(null);
    const nullStore = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        store: null,
      },
    });
    await flushPromises();
    await nullStore.find('[data-testid="o-properties-definition-save"]').trigger('click');
    await flushPromises();
    // loadError takes UI precedence over saveError when store resolves to null.
    const saveErrEl = nullStore.find('[data-testid="o-properties-definition-save-error"]');
    const loadErrEl = nullStore.find('[data-testid="o-properties-definition-error"]');
    const err = (saveErrEl.exists() ? saveErrEl.text() : '') || (loadErrEl.exists() ? loadErrEl.text() : '');
    expect(err).toContain('PropertyDefinition store is unavailable');
    nullStore.unmount();

    // non-Error throw on save catch path
    const bareSave = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        store: {
          Search: vi.fn(async () => []),
          Create: vi.fn(async () => {
            throw 'bare-save-fail';
          }),
          UpdateById: vi.fn(),
        },
      },
    });
    await flushPromises();
    await bareSave.find('[data-testid="o-properties-definition-add"]').trigger('click');
    await nextTick();
    await bareSave.find('[data-testid="o-properties-definition-name"]').setValue('x');
    await bareSave.find('[data-testid="o-properties-definition-save"]').trigger('click');
    await flushPromises();
    expect(bareSave.find('[data-testid="o-properties-definition-save-error"]').text()).toContain('bare-save-fail');
    bareSave.unmount();

    warn.mockRestore();
  });

  it('ignores add/save while loading or saving via setupState guards', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    let resolveSearch: (v: any) => void = () => undefined;
    const Search = vi.fn(
      () =>
        new Promise(resolve => {
          resolveSearch = resolve;
        })
    );
    const Create = vi.fn();
    const wrapper = mount(OPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        store: { Search, Create, UpdateById: vi.fn() },
      },
    });
    await nextTick();
    const state = (wrapper.vm as any).$.setupState;
    expect(state.loading).toBe(true);
    // disabled buttons suppress click — exercise onAdd/onSave guards directly
    state.onAdd();
    await state.onSave();
    expect(Create).not.toHaveBeenCalled();
    expect(state.drafts.length).toBe(0);
    Search.mockResolvedValue([]);
    resolveSearch([]);
    await flushPromises();

    // saving guard: hang Create then call onAdd/onSave again
    let resolveCreate: (v: any) => void = () => undefined;
    Create.mockImplementationOnce(
      () =>
        new Promise(resolve => {
          resolveCreate = resolve;
        })
    );
    state.onAdd();
    await nextTick();
    state.drafts[0].name = 'hang';
    const saveP = state.onSave();
    await nextTick();
    expect(state.saving).toBe(true);
    state.onAdd();
    await state.onSave();
    expect(state.drafts).toHaveLength(1);
    resolveCreate({ Id: 'hang-1' });
    await saveP;
    await flushPromises();
    wrapper.unmount();
    warn.mockRestore();
  });
});
