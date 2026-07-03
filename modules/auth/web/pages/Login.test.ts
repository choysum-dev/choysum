// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';

let mockAuthStore: any;

// Mock useAuthStore.
vi.mock('../stores/auth', () => {
  return {
    useAuthStore: () => mockAuthStore,
  };
});

// Mock vue-router.
const mockRouterReplace = vi.fn();
vi.mock('vue-router', () => {
  return {
    useRouter: () => ({ replace: mockRouterReplace }),
    useRoute: () => ({ query: {} }),
  };
});

// Mock pinia storeToRefs.
vi.mock('pinia', () => {
  return {
    storeToRefs: (store: any) => ({
      loading: { value: store.loading?.value ?? false },
      isAuthenticated: { value: store.isAuthenticated?.value ?? false },
    }),
  };
});

// Mock OPage wrapper.
vi.mock('@/web/web/components/page/OPage.vue', () => {
  return {
    default: {
      name: 'OPage',
      template: '<div class="o-page"><slot /></div>',
      props: ['loading'],
    },
  };
});

// Mock element-plus.
vi.mock('element-plus', () => {
  return {
    ElForm: { name: 'ElForm', template: '<form @submit.prevent="$emit(\'submit\')"><slot /></form>', props: ['model', 'rules', 'labelPosition'], emits: ['submit'] },
    ElFormItem: { name: 'ElFormItem', template: '<div><slot /></div>', props: ['prop', 'label'] },
    ElInput: { name: 'ElInput', template: '<input />', props: ['modelValue', 'placeholder', 'prefixIcon', 'type', 'autocomplete', 'showPassword'], emits: ['update:modelValue'] },
    ElButton: { name: 'ElButton', template: '<button><slot /></button>', props: ['type', 'loading', 'nativeType'] },
    ElCheckbox: { name: 'ElCheckbox', template: '<input type="checkbox" />', props: ['modelValue'], emits: ['update:modelValue'] },
    ElAlert: { name: 'ElAlert', template: '<div><slot /></div>', props: ['title', 'type', 'closable'], emits: ['close'] },
    ElCard: { name: 'ElCard', template: '<div><slot name="header" /><slot /></div>', props: ['shadow'] },
  };
});

// Mock element-plus icons.
vi.mock('@element-plus/icons-vue', () => {
  return { User: 'user-icon', Lock: 'lock-icon' };
});

function buildMockAuthStore(overrides: Record<string, unknown> = {}) {
  return {
    loading: { value: false },
    isAuthenticated: { value: false },
    ensureAuthReady: vi.fn(async () => undefined),
    login: vi.fn(),
    ...overrides,
  };
}

describe('Login.vue onMounted', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockRouterReplace.mockReset();
    vi.resetModules();
  });

  it('calls ensureAuthReady on mount', async () => {
    mockAuthStore = buildMockAuthStore();

    const { default: Login } = await import('../pages/Login.vue');
    mount(Login);

    expect(mockAuthStore.ensureAuthReady).toHaveBeenCalledTimes(1);
  });

  it('redirects to home when already authenticated after init', async () => {
    mockAuthStore = buildMockAuthStore({
      isAuthenticated: { value: true },
    });

    const { default: Login } = await import('../pages/Login.vue');
    mount(Login);

    expect(mockAuthStore.ensureAuthReady).toHaveBeenCalledTimes(1);
    // Wait for the async onMounted to complete.
    await vi.waitFor(() => {
      expect(mockRouterReplace).toHaveBeenCalledWith('/');
    });
  });

  it('does not redirect when ensureAuthReady throws', async () => {
    mockAuthStore = buildMockAuthStore({
      isAuthenticated: { value: false },
      ensureAuthReady: vi.fn(async () => {
        throw new Error('init failed');
      }),
    });

    const { default: Login } = await import('../pages/Login.vue');
    mount(Login);

    expect(mockAuthStore.ensureAuthReady).toHaveBeenCalledTimes(1);
    // ensureAuthReady threw, but the error was caught.
    // isAuthenticated is still false, so no redirect.
    await vi.waitFor(() => {
      // No redirect should happen.
      expect(mockRouterReplace).not.toHaveBeenCalled();
    });
  });
});
