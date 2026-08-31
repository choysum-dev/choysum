// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { config, shallowMount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import { nextTick, reactive } from 'vue';
import { appRoutes } from '@/auth/web/route/routes';

config.global.renderStubDefaultSlot = true;

const push = vi.fn();

vi.mock('vue-router', () => ({
  useRouter: () => ({ push, replace: vi.fn() }),
  useRoute: () => ({ fullPath: '/auth/record-rules', params: { id: 'rr-1' }, path: '/auth/record-rules', query: {} }),
}));

vi.mock('@/auth/web/composables/usePermission', () => ({
  usePermission: () => ({
    canRoute: () => true,
    hasAction: () => true,
  }),
}));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: vi.fn(() => fakeStore),
}));

vi.mock('@/web/web/stores/storeScopeManager', () => ({
  useScopeManager: () => ({ menuScopeManager: {} }),
}));

const fakeStore = {
  state: { queryState: {}, result: undefined, selection: [], planCache: new Map() },
  setContext: vi.fn(),
  getContext: vi.fn(),
  withContext: vi.fn(),
  fieldsMetadata: {},
};

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: {} },
});

/** Provide `$route` for page templates that still use `$route.fullPath` as :key. */
const routeGlobal = {
  install(app: { config: { globalProperties: Record<string, unknown> } }) {
    app.config.globalProperties.$route = {
      fullPath: '/auth/record-rules',
      params: { id: 'id-1' },
      path: '/auth/record-rules',
      query: {},
    };
  },
};

const ACCESS_RULE_ROUTE_NAMES = new Set([
  'RecordRuleList',
  'RecordRuleDetail',
  'RecordRuleCreate',
  'FieldRuleList',
  'FieldRuleDetail',
  'FieldRuleCreate',
  'MethodAccessList',
  'MethodAccessDetail',
  'MethodAccessCreate',
  'UiResourceGrantList',
  'UiResourceGrantDetail',
  'UiResourceGrantCreate',
]);

const pageModules = import.meta.glob('../pages/Role{RecordRule,FieldRule,MethodAccess,UiResource}*.vue', {
  eager: true,
}) as Record<string, { default: object }>;

const listViewModules = import.meta.glob('./Role{RecordRule,FieldRule,MethodAccess,UiResource}*ListView.vue', {
  eager: true,
}) as Record<string, { default: object }>;

const formViewModules = import.meta.glob('./Role{RecordRule,FieldRule,MethodAccess,UiResource}*FormView.vue', {
  eager: true,
}) as Record<string, { default: object }>;

describe('Access Rules admin coverage (PR-C-5)', () => {
  beforeEach(() => {
    push.mockReset();
  });

  it('exercises Access Rules route components without page prop injection', async () => {
    const accessRoutes = appRoutes.filter(r => ACCESS_RULE_ROUTE_NAMES.has(String(r.name)));
    expect(accessRoutes.length).toBe(12);

    for (const route of accessRoutes) {
      const comp = route.component as (() => Promise<unknown>) | undefined;
      expect(typeof comp).toBe('function');
      const loaded = await (comp as () => Promise<{ default: unknown }>)();
      expect(loaded?.default).toBeTruthy();

      // Detail/create pages resolve record id from the route inside OFormView.
      expect(route.props == null || route.props === false).toBe(true);
    }
  });

  it('mounts Access Rules page shells', () => {
    const paths = Object.keys(pageModules);
    expect(paths.length).toBe(8);

    for (const [path, mod] of Object.entries(pageModules)) {
      const wrapper = shallowMount(mod.default as any, {
        props: {},
        global: {
          plugins: [i18n, routeGlobal],
          stubs: {
            OPage: { name: 'OPage', template: '<div class="opage-stub"><slot /></div>' },
            RoleRecordRuleFormView: true,
            RoleRecordRuleListView: true,
            RoleFieldRuleFormView: true,
            RoleFieldRuleListView: true,
            RoleMethodAccessFormView: true,
            RoleMethodAccessListView: true,
            RoleUiResourceFormView: true,
            RoleUiResourceListView: true,
          },
        },
      });
      expect(wrapper.exists()).toBe(true);
      wrapper.unmount();
    }
  });

  it('mounts AudienceHints and toggles grant-everyone warning', async () => {
    const { default: Hints } = await import('./RoleRecordRuleAudienceHints.vue');
    const formRoot = reactive<{ draft: Record<string, any> | null }>({ draft: null });

    const wrapper = shallowMount(Hints as any, {
      global: {
        plugins: [i18n],
        provide: { 'form-root': formRoot },
        stubs: { ElAlert: { name: 'ElAlert', template: '<div class="el-alert" :class="$attrs.class"><slot /></div>', props: ['title', 'type', 'description', 'closable', 'showIcon'] } },
      },
    });

    expect(wrapper.find('.rr-audience-hints__warn').exists()).toBe(false);

    formRoot.draft = { Kind: 'grant', RoleId: null };
    await nextTick();
    expect(wrapper.find('.rr-audience-hints__warn').exists()).toBe(true);

    formRoot.draft = { Kind: 'restrict', RoleId: null };
    await nextTick();
    expect(wrapper.find('.rr-audience-hints__warn').exists()).toBe(false);

    formRoot.draft = { Kind: 'grant', RoleId: { Id: 'role-1' } };
    await nextTick();
    expect(wrapper.find('.rr-audience-hints__warn').exists()).toBe(false);

    wrapper.unmount();
  });

  it('covers list onRowClick navigation', async () => {
    for (const [path, mod] of Object.entries(listViewModules)) {
      const wrapper = shallowMount(mod.default as any, {
        props: { store: fakeStore as any },
        global: {
          plugins: [i18n],
          stubs: {
            OListView: {
              name: 'OListView',
              template: '<div class="olist-stub" />',
              emits: ['row-click'],
            },
          },
          directives: { action: { mounted() {}, updated() {} } },
        },
      });

      const list = wrapper.findComponent({ name: 'OListView' });
      expect(list.exists()).toBe(true);
      await list.vm.$emit('row-click', { row: { Id: 'row-1' } });
      expect(push).toHaveBeenCalled();
      const dest = String(push.mock.calls.at(-1)?.[0] ?? '');
      expect(dest).toMatch(/\/auth\/.+\//);
      expect(dest).toContain('row-1');
      wrapper.unmount();
      push.mockClear();
      void path;
    }
  });

  it('covers form onRoleValueClick navigation', async () => {
    for (const [path, mod] of Object.entries(formViewModules)) {
      const wrapper = shallowMount(mod.default as any, {
        props: { store: fakeStore as any },
        global: {
          plugins: [i18n],
          stubs: {
            OFormView: {
              name: 'OFormView',
              template: '<div class="oform-stub"><slot /></div>',
            },
            OManyToOneField: {
              name: 'OManyToOneField',
              template: '<div class="m2o-stub" />',
              emits: ['value-click'],
            },
          },
          directives: { action: { mounted() {}, updated() {} } },
        },
      });

      const m2o = wrapper.findComponent({ name: 'OManyToOneField' });
      expect(m2o.exists()).toBe(true);
      // Cover payload?.id branches: missing payload, empty id, whitespace, real id.
      await m2o.vm.$emit('value-click', undefined);
      await m2o.vm.$emit('value-click', null);
      await m2o.vm.$emit('value-click', {});
      await m2o.vm.$emit('value-click', { id: null });
      await m2o.vm.$emit('value-click', { id: '  ' });
      expect(push).not.toHaveBeenCalled();
      await m2o.vm.$emit('value-click', { id: 'role-9' });
      expect(push).toHaveBeenCalledWith('/auth/roles/role-9');
      wrapper.unmount();
      push.mockClear();
      void path;
    }
  });
});
