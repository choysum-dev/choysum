// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { computed, defineComponent, h, nextTick, ref } from 'vue';
import { Comment, Fragment, Text } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import type { UseField } from '@/web/web/composables/useField';
import OStatInfo from './OStatInfo.vue';
import OButtonBox from './OButtonBox.vue';
import { resolveStatDisplayValue, slotHasContent } from './ostatinfo_helpers';

const push = vi.fn();
vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}));

vi.mock('@/web/web/composables/useField', async importOriginal => {
  const mod = await importOriginal<typeof import('@/web/web/composables/useField')>();
  return {
    ...mod,
    useField: vi.fn(mod.useField),
  };
});

import { useField } from '@/web/web/composables/useField';

function makeRelationBinding(value: unknown): UseField {
  const field = ref(value);
  return {
    env: {
      isForm: true,
      isEditMode: false,
      viewMode: 'display',
      fieldPrefix: null,
    },
    prop: 'Users',
    meta: undefined as any,
    fieldRef: () => field as any,
    fieldRefOf: () => field as any,
    recordRef: () => computed(() => ({ Users: field.value })) as any,
    registerFields: () => {},
    store: { storeId: 'test' } as any,
    asView: () => ({ fieldValue: () => field }) as any,
  } as UseField;
}

describe('ostatinfo_helpers', () => {
  it('prefers explicit value over relation length', () => {
    expect(resolveStatDisplayValue({ value: 3, relationValue: [1, 2] })).toBe(3);
    expect(resolveStatDisplayValue({ value: 0, relationValue: [1] })).toBe(0);
  });

  it('uses relation array length when value is absent', () => {
    expect(resolveStatDisplayValue({ relationValue: ['a', 'b'] })).toBe(2);
    expect(resolveStatDisplayValue({ relationValue: [] })).toBe(0);
  });

  it('falls back to em dash (not 0) when unloaded', () => {
    expect(resolveStatDisplayValue({})).toBe('—');
    expect(resolveStatDisplayValue({ relationValue: null })).toBe('—');
    expect(resolveStatDisplayValue({ value: null })).toBe('—');
    expect(resolveStatDisplayValue({ emptyValue: 0 })).toBe(0);
  });

  it('detects empty vs meaningful slot trees', () => {
    expect(slotHasContent(null)).toBe(false);
    expect(slotHasContent(undefined)).toBe(false);
    expect(slotHasContent([])).toBe(false);
    expect(slotHasContent([h(Comment, 'x')])).toBe(false);
    expect(slotHasContent([h(Text, '   ')])).toBe(false);
    expect(slotHasContent([h(Text)])).toBe(false); // children null/undefined → ?? ''
    expect(slotHasContent([h(Text, 'hi')])).toBe(true);
    expect(slotHasContent([h('div')])).toBe(true);
    expect(slotHasContent([h(Fragment, [h(Comment), h('span')])])).toBe(true);
    // Non-VNode entries must not count as content (isMeaningfulVNode false branches).
    expect(slotHasContent([null, undefined, 42, 'plain'])).toBe(false);
    // Fragment with non-array children is empty.
    expect(slotHasContent([{ type: Fragment, children: 'x' }])).toBe(false);
  });
});

describe('OStatInfo', () => {
  beforeEach(() => {
    push.mockReset();
    vi.mocked(useField).mockClear();
  });

  it('renders value and label; emits click', async () => {
    const wrapper = mount(OStatInfo as any, {
      props: { value: 5, label: 'Users' },
    });
    expect(wrapper.find('.o-stat-info__value').text()).toBe('5');
    expect(wrapper.find('.o-stat-info__label').text()).toBe('Users');
    expect(wrapper.find('.o-stat-info__icon').exists()).toBe(false);
    await wrapper.trigger('click');
    expect(wrapper.emitted('click')?.length).toBe(1);
    expect(push).not.toHaveBeenCalled();
  });

  it('renders icon when icon prop is set', () => {
    const IconStub = defineComponent({
      name: 'IconStub',
      template: '<span class="icon-stub" />',
    });
    const wrapper = mount(OStatInfo as any, {
      props: { value: 1, label: 'Users', icon: IconStub },
      global: {
        stubs: {
          ElIcon: { template: '<i class="el-icon-stub"><slot /></i>' },
        },
      },
    });
    expect(wrapper.find('.o-stat-info__icon').exists()).toBe(true);
    expect(wrapper.find('.icon-stub').exists()).toBe(true);
  });

  it('uses relation length via store+prop when value omitted', async () => {
    vi.mocked(useField).mockReturnValueOnce(makeRelationBinding([{ Id: '1' }, { Id: '2' }]));
    const wrapper = mount(OStatInfo as any, {
      props: { store: { storeId: 's' }, prop: 'Users', label: 'Users' },
    });
    await flushPromises();
    expect(wrapper.find('.o-stat-info__value').text()).toBe('2');
  });

  it('skips useField when only store or only prop is set', () => {
    mount(OStatInfo as any, {
      props: { store: { storeId: 's' }, label: 'Users', value: 1 },
    });
    mount(OStatInfo as any, {
      props: { prop: 'Users', label: 'Users', value: 1 },
    });
    expect(vi.mocked(useField)).not.toHaveBeenCalled();
  });

  it('shows em dash when relation is not an array', async () => {
    vi.mocked(useField).mockReturnValueOnce(makeRelationBinding(undefined));
    const wrapper = mount(OStatInfo as any, {
      props: { store: { storeId: 's' }, prop: 'Users', label: 'Users' },
    });
    expect(wrapper.find('.o-stat-info__value').text()).toBe('—');
  });

  it('router.push(to) after emit on click', async () => {
    const wrapper = mount(OStatInfo as any, {
      props: { value: 1, label: 'Go', to: { name: 'users' } },
    });
    await wrapper.trigger('click');
    expect(wrapper.emitted('click')?.length).toBe(1);
    expect(push).toHaveBeenCalledWith({ name: 'users' });
  });

  it('does not render when visible=false; ignores click when disabled', async () => {
    const hidden = mount(OStatInfo as any, {
      props: { value: 1, label: 'X', visible: false },
    });
    expect(hidden.find('.o-stat-info').exists()).toBe(false);

    const disabled = mount(OStatInfo as any, {
      props: { value: 1, label: 'X', disabled: true, to: { name: 'users' } },
    });
    // Native disabled suppresses click; clear it so onClick's props.disabled guard runs.
    const btn = disabled.find('button');
    (btn.element as HTMLButtonElement).disabled = false;
    await btn.trigger('click');
    expect(disabled.emitted('click')).toBeUndefined();
    expect(push).not.toHaveBeenCalled();
  });
});

describe('OButtonBox', () => {
  it('does not render root when default slot is empty', () => {
    const empty = mount(OButtonBox as any, { slots: {} });
    expect(empty.find('.o-button-box').exists()).toBe(false);

    const withChild = mount(OButtonBox as any, {
      slots: { default: () => h(OStatInfo as any, { value: 1, label: 'A' }) },
    });
    expect(withChild.find('.o-button-box').exists()).toBe(true);
    expect(withChild.find('.o-stat-info__value').text()).toBe('1');
  });

  it('remounts shell when slot children appear later', async () => {
    const show = ref(false);
    const Host = defineComponent({
      components: { OButtonBox, OStatInfo },
      setup() {
        return { show };
      },
      template: `
        <OButtonBox>
          <OStatInfo v-if="show" :value="1" label="A" />
        </OButtonBox>
      `,
    });
    const wrapper = mount(Host);
    expect(wrapper.find('.o-button-box').exists()).toBe(false);
    show.value = true;
    await nextTick();
    expect(wrapper.find('.o-button-box').exists()).toBe(true);
    expect(wrapper.find('.o-stat-info__value').text()).toBe('1');
  });
});

describe('OFormView #button-box slot placement', () => {
  it('declares button-box after statusbar in template and defineSlots', () => {
    const src = readFileSync(resolve(__dirname, './OFormView.vue'), 'utf8');
    const statusbar = src.indexOf('<slot name="statusbar"');
    const buttonBox = src.indexOf('<slot name="button-box"');
    const headerRight = src.indexOf('<slot name="header-right"');
    expect(statusbar).toBeGreaterThan(-1);
    expect(buttonBox).toBeGreaterThan(-1);
    expect(headerRight).toBeGreaterThan(-1);
    expect(statusbar).toBeLessThan(buttonBox);
    expect(buttonBox).toBeLessThan(headerRight);
    expect(src).toContain("'button-box'(): any");
  });
});
