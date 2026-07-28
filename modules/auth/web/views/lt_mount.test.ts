// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { config, shallowMount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import { nextTick } from 'vue';

// Stubbed OFormView/OListView must still render default slots so template
// field lines (the P6 :label removals) are exercised for coverage.
config.global.renderStubDefaultSlot = true;

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  useRoute: () => ({ params: {}, query: {}, path: '/' }),
}));

vi.mock('@/auth/web/composables/usePermission', () => ({
  usePermission: () => ({
    canRoute: () => true,
    hasAction: () => true,
  }),
}));

const fakeStore = {
  state: { queryState: {}, result: undefined, selection: [], planCache: new Map() },
  setContext: vi.fn(),
  getContext: vi.fn(),
  withContext: vi.fn(),
};

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: {} },
});

const viewModules = import.meta.glob('./*View.vue', { eager: true }) as Record<
  string,
  { default: object }
>;

function mountView(mod: object) {
  return shallowMount(mod as any, {
    props: { store: fakeStore as any },
    global: {
      plugins: [i18n],
      stubs: true,
      // Views may use v-action; coverage mounts do not install the real directive.
      directives: {
        action: {
          mounted() {},
          updated() {},
        },
      },
    },
  });
}

describe('auth view _lt setup coverage', () => {
  for (const [path, mod] of Object.entries(viewModules)) {
    it(`mounts ${path} so createTranslate/_lt run`, () => {
      const wrapper = mountView(mod.default);
      expect(wrapper.exists()).toBe(true);
      wrapper.unmount();
    });
  }
});

describe('RoleFormView UI Requires inspector coverage', () => {
  it('exercises inspect helpers, icon branches, and Requires panel', async () => {
    const mod = viewModules['./RoleFormView.vue'];
    expect(mod?.default).toBeTruthy();
    const wrapper = shallowMount(mod.default as any, {
      props: { store: fakeStore as any },
      global: {
        plugins: [i18n],
        stubs: {
          OManyToManyRefTreeField: {
            name: 'OManyToManyRefTreeField',
            template:
              '<div class="tree-stub"><slot name="node" :row="sample" :label="\'Label\'" /></div>',
            setup() {
              return {
                sample: {
                  Id: 'uir-slot',
                  Type: 'MENU',
                  Title: 'Slot Menu',
                  Requires: ['rpc:/auth.User/Browse'],
                },
              };
            },
          },
        },
        directives: {
          action: {
            mounted() {},
            updated() {},
          },
        },
      },
    });
    const vm = wrapper.vm as any;

    expect(vm.resolveUiResourceTypeIcon('MENU')).toBeTruthy();
    expect(vm.resolveUiResourceTypeIcon('ROUTE')).toBeTruthy();
    expect(vm.resolveUiResourceTypeIcon('ACTION')).toBeTruthy();
    expect(vm.resolveUiResourceTypeIcon('OTHER')).toBeTruthy();
    expect(vm.resolveUiResourceTypeIcon()).toBeTruthy();

    // Cover resolveUiResourceLabel branches (explicit label + Title/Name/Id/empty).
    expect(vm.resolveUiResourceLabel({ Title: 'T' }, 'LabelWins')).toContain('LabelWins');
    expect(vm.resolveUiResourceLabel({ Title: 'Only Title' })).toContain('Only Title');
    expect(vm.resolveUiResourceLabel({ Name: 'Only Name' })).toContain('Only Name');
    expect(vm.resolveUiResourceLabel({ Id: 'only-id' })).toContain('only-id');
    expect(vm.resolveUiResourceLabel({})).toBe('');
    expect(vm.resolveUiResourceLabel(undefined)).toBe('');
    expect(vm.resolveUiResourceLabel({ TitleText: null, Title: '' })).toBe('');

    vm.inspectUiResource({ Id: 'only-id' });
    expect(String(vm.inspectedUiResourceLabel)).toContain('only-id');
    vm.inspectUiResource({ Title: 'Only Title', Requires: ['rpc:/x/y'] });
    expect(String(vm.inspectedUiResourceLabel)).toContain('Only Title');
    vm.inspectUiResource({});
    expect(String(vm.inspectedUiResourceLabel)).toBe('');
    vm.inspectUiResource(null);
    expect(String(vm.inspectedUiResourceLabel)).toBe('');

    vm.activeTab = 'ui_permissions';
    await nextTick();

    // Cover template v-model update handlers (not just direct ref writes).
    const tabs = wrapper.findComponent({ name: 'ElTabs' });
    if (tabs.exists()) {
      await tabs.vm.$emit('update:modelValue', 'ui_permissions');
      await nextTick();
      await tabs.vm.$emit('update:modelValue', 'advanced');
      await nextTick();
      await tabs.vm.$emit('update:modelValue', 'users');
      await nextTick();
      await tabs.vm.$emit('update:modelValue', 'ui_permissions');
      await nextTick();
    }

    const btn = wrapper.find('button.rfv-ui-resource-node');
    expect(btn.exists()).toBe(true);
    await btn.trigger('click');
    await nextTick();
    expect(vm.inspectedUiResourceId).toBe('uir-slot');
    expect(vm.inspectedRequires).toEqual(['rpc:/auth.User/Browse']);
    expect(wrapper.text()).toContain('rpc:/auth.User/Browse');

    vm.inspectUiResource(null);
    expect(vm.inspectedUiResource).toBeNull();
    expect(vm.inspectedUiResourceId).toBe('');
    expect(vm.inspectedRequires).toEqual([]);

    vm.inspectUiResource('not-a-row');
    expect(vm.inspectedUiResource).toBeNull();

    vm.inspectUiResource({ Id: 'uir-2', Name: 'Empty', requires: [] });
    await nextTick();
    expect(vm.inspectedRequires).toEqual([]);
    expect(wrapper.text()).toMatch(/No Requires/i);

    if (tabs.exists()) {
      await tabs.vm.$emit('update:modelValue', 'advanced');
      await nextTick();
    } else {
      vm.activeTab = 'advanced';
      await nextTick();
    }
    vm.advancedPanels = 'record_rules';
    await nextTick();
    const collapse = wrapper.findComponent({ name: 'ElCollapse' });
    if (collapse.exists()) {
      await collapse.vm.$emit('update:modelValue', 'record_rules');
      await nextTick();
      await collapse.vm.$emit('update:modelValue', '');
      await nextTick();
    }
    expect(wrapper.text()).toMatch(/Record Rules|matching grant|deny-default/i);

    wrapper.unmount();
  });
});
