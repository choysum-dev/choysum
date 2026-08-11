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

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({
      _t: (msg: string) => msg,
    }),
  };
});

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
            h('select', {
              ...attrs,
              value: props.modelValue == null ? '' : String(props.modelValue),
              disabled: props.disabled || undefined,
              onChange: (e: Event) => emit('update:modelValue', (e.target as HTMLSelectElement).value),
            }, slots.default?.());
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
  it('round-trips drafts and builds scope conditions', () => {
    expect(emptyDraftItem().type).toBe('char');
    expect(definitionItemsToDrafts(null)).toEqual([]);
    const drafts = definitionItemsToDrafts([
      { name: 'code', type: 'char', string: 'Code', default: 'A' },
      { name: 'kind', type: 'selection', selection: [['a', 'A']] },
      { name: '', type: 'char' },
    ]);
    expect(drafts).toHaveLength(2);
    expect(drafts[1]?.selectionText).toContain('a');

    expect(draftsToDefinitionItems(drafts)).toEqual([
      { name: 'code', type: 'char', string: 'Code', default: 'A' },
      { name: 'kind', type: 'selection', selection: [['a', 'A']] },
    ]);

    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 'x', type: 'html' }])
    ).toThrow(/unsupported property type/);
    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '' }])
    ).toThrow(/requires selection JSON/);

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
  });
});

describe('OPropertiesDefinitionEditor', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
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
});
