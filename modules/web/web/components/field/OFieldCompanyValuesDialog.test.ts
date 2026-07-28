// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';

import OFieldCompanyValuesDialog from './OFieldCompanyValuesDialog.vue';

const { companyResponses } = vi.hoisted(() => ({
  companyResponses: {
    rows: [
      { Id: 'comp_main', DisplayName: 'Main Company' },
      { Id: 'comp_eu', DisplayName: 'EU Company' },
    ] as Array<{ Id: string; DisplayName: string }>,
  },
}));

const { authStoreState } = vi.hoisted(() => ({
  authStoreState: {
    metadata: {
      allowedCompanyIds: ['comp_main', 'comp_eu'] as string[],
      enabledCompanyIds: ['comp_main'] as string[],
      activeCompanyId: 'comp_main',
    } as Record<string, unknown>,
    throwOnAccess: false,
  },
}));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: (model: string) => {
    if (model === 'base.Company') {
      return {
        Search: async () => companyResponses.rows,
      };
    }
    return {};
  },
}));

vi.mock('@/auth/web/stores/auth', () => ({
  useAuthStore: () => {
    if (authStoreState.throwOnAccess) throw new Error('no pinia');
    return { identity: { metadata: authStoreState.metadata } };
  },
}));

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  return {
    ...actual,
    ElMessage: { success: vi.fn(), error: vi.fn() },
  };
});

const dialogStubs = {
  'el-dialog': {
    props: ['modelValue', 'title'],
    template:
      '<div class="dialog" :data-title="title"><button class="emit-closed" type="button" @click="$emit(\'closed\')" /><slot /><slot name="footer" /></div>',
    emits: ['opened', 'closed', 'update:modelValue'],
    mounted() {
      this.$emit('opened');
    },
  },
  'el-form': { template: '<form><slot /></form>' },
  'el-form-item': { props: ['label'], template: '<div class="item" :data-label="label"><slot /></div>' },
  'el-input': {
    props: ['modelValue', 'maxlength', 'showWordLimit'],
    emits: ['update:modelValue'],
    template:
      '<input class="input" :value="modelValue" :data-maxlength="maxlength" :data-word-limit="showWordLimit" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
  'el-button': {
    props: ['loading', 'type'],
    template: '<button class="btn" @click="$emit(\'click\')"><slot /></button>',
  },
};

const loadingDirective = {
  mounted() {},
  updated() {},
};

function mountDialog(props: Record<string, unknown>) {
  return mount(OFieldCompanyValuesDialog, {
    props: props as any,
    global: {
      stubs: dialogStubs,
      directives: { loading: loadingDirective },
    },
  });
}

async function flushOpen() {
  await nextTick();
  await Promise.resolve();
  await nextTick();
}

function inputIndexByLabel(wrapper: ReturnType<typeof mountDialog>, label: string): number {
  return wrapper.findAll('.item').findIndex(el => el.attributes('data-label') === label);
}

describe('OFieldCompanyValuesDialog', () => {
  it('loads allowed companies and saves UpdateFieldCompanyValues patch', async () => {
    authStoreState.throwOnAccess = false;
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main Company' },
      { Id: 'comp_eu', DisplayName: 'EU Company' },
    ];
    const UpdateFieldCompanyValues = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({ Cost: '11.5' }));
    const GetFieldCompanyValues = vi.fn(async () => ({ comp_main: '12.5', comp_eu: '11.0' }));

    const wrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues, Browse } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
      fieldLabel: 'Cost',
    });

    await flushOpen();

    expect(GetFieldCompanyValues).toHaveBeenCalledWith('prod-1', 'Cost');
    const labels = wrapper.findAll('.item').map(el => el.attributes('data-label'));
    expect(labels).toContain('Main Company');
    expect(labels).toContain('EU Company');

    const euIdx = inputIndexByLabel(wrapper, 'EU Company');
    await wrapper.findAll('.input')[euIdx]!.setValue('11.5');
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();

    expect(UpdateFieldCompanyValues).toHaveBeenCalledWith('prod-1', 'Cost', { comp_eu: '11.5' });
    expect(Browse).toHaveBeenCalled();
    expect(wrapper.emitted('saved')?.[0]?.[0]).toBe('11.5');
  });

  it('clears any company value with false delete sentinel', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main Company' },
      { Id: 'comp_eu', DisplayName: 'EU Company' },
    ];
    const UpdateFieldCompanyValues = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({ Cost: '12.5' }));
    const GetFieldCompanyValues = vi.fn(async () => ({ comp_main: '12.5', comp_eu: '11.0' }));

    const wrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues, Browse } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
      fieldLabel: 'Cost',
    });

    await flushOpen();
    const mainIdx = inputIndexByLabel(wrapper, 'Main Company');
    await wrapper.findAll('.input')[mainIdx]!.setValue('');
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();

    expect(UpdateFieldCompanyValues).toHaveBeenCalledWith('prod-1', 'Cost', { comp_main: false });
    expect(wrapper.emitted('saved')?.[0]?.[0]).toBe('12.5');
  });

  it('seeds the current-company row from draftValue on open and saves it as dirty', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main Company' },
      { Id: 'comp_eu', DisplayName: 'EU Company' },
    ];
    const UpdateFieldCompanyValues = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({ Cost: '99' }));
    const GetFieldCompanyValues = vi.fn(async () => ({
      comp_main: '12.5',
      comp_eu: '11.0',
    }));

    const wrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues, Browse } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
      fieldLabel: 'Cost',
      draftValue: '99',
      draftCompanyId: 'comp_main',
    });

    await flushOpen();
    const mainIdx = inputIndexByLabel(wrapper, 'Main Company');
    expect(wrapper.findAll('.input')[mainIdx]!.element).toMatchObject({ value: '99' });

    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();

    expect(UpdateFieldCompanyValues).toHaveBeenCalledWith('prod-1', 'Cost', {
      comp_main: '99',
    });
  });

  it('shows load error and keeps empty rows when GetFieldCompanyValues fails', async () => {
    const { ElMessage } = await import('element-plus');
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main'],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    };
    const GetFieldCompanyValues = vi.fn(async () => {
      throw new Error('load boom');
    });
    const wrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues: vi.fn(), Browse: vi.fn() } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
      fieldLabel: 'Cost',
    });
    await flushOpen();
    expect(ElMessage.error).toHaveBeenCalled();
    expect(wrapper.findAll('.input')).toHaveLength(0);
  });

  it('falls back to enabled ∪ map keys when allowedCompanyIds is empty', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: [],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main Company' },
      { Id: 'comp_eu', DisplayName: 'EU Company' },
    ];
    const GetFieldCompanyValues = vi.fn(async () => ({ comp_main: '1', comp_eu: '2' }));
    const wrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues: vi.fn(), Browse: vi.fn(async () => ({})) } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
    });
    await flushOpen();
    const labels = wrapper.findAll('.item').map(el => el.attributes('data-label'));
    expect(labels).toContain('Main Company');
    expect(labels).toContain('EU Company');
    expect(wrapper.get('.dialog').attributes('data-title')).toMatch(/Company values/);
  });

  it('uses company id as label fallback, respects maxLength, and cancels', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: '' },
      { Id: 'comp_eu', DisplayName: 'EU Company' },
    ];
    const UpdateFieldCompanyValues = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({ Cost: '' }));
    const GetFieldCompanyValues = vi.fn(async () => ({ comp_main: 'Hello', comp_eu: '你好' }));

    const wrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues, Browse } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
      maxLength: 40,
    });
    await flushOpen();
    expect(wrapper.get('.input').attributes('data-maxlength')).toBe('40');
    const labels = wrapper.findAll('.item').map(el => el.attributes('data-label'));
    expect(labels).toContain('comp_main');

    const wrapper2 = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues: vi.fn(), Browse: vi.fn() } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
      fieldLabel: 'Cost',
    });
    await flushOpen();
    await wrapper2.findAll('.btn')[0]!.trigger('click');
    expect(wrapper2.emitted('update:modelValue')?.at(-1)?.[0]).toBe(false);
  });

  it('skips UpdateFieldCompanyValues when nothing is dirty but still browses and closes', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main Company' },
      { Id: 'comp_eu', DisplayName: 'EU Company' },
    ];
    const UpdateFieldCompanyValues = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({ Cost: '12.5' }));
    const GetFieldCompanyValues = vi.fn(async () => ({ comp_main: '12.5', comp_eu: '11.0' }));
    const wrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues, Browse } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
      fieldLabel: 'Cost',
    });
    await flushOpen();
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateFieldCompanyValues).not.toHaveBeenCalled();
    expect(Browse).toHaveBeenCalled();
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toBe(false);
  });

  it('surfaces save errors without closing the dialog', async () => {
    const { ElMessage } = await import('element-plus');
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main Company' },
      { Id: 'comp_eu', DisplayName: 'EU Company' },
    ];
    const UpdateFieldCompanyValues = vi.fn(async () => {
      throw new Error('save boom');
    });
    const Browse = vi.fn(async () => ({ Cost: '12.5' }));
    const GetFieldCompanyValues = vi.fn(async () => ({ comp_main: '12.5', comp_eu: '11.0' }));
    const wrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues, Browse } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
      fieldLabel: 'Cost',
    });
    await flushOpen();
    const euIdx = inputIndexByLabel(wrapper, 'EU Company');
    await wrapper.findAll('.input')[euIdx]!.setValue('9');
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(ElMessage.error).toHaveBeenCalled();
    expect(wrapper.emitted('update:modelValue')?.some(args => args[0] === false)).toBeFalsy();
  });

  it('resolves draftCompanyId from auth activeCompanyId when prop is omitted', async () => {
    authStoreState.throwOnAccess = false;
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_eu',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main Company' },
      { Id: 'comp_eu', DisplayName: 'EU Company' },
    ];
    const GetFieldCompanyValues = vi.fn(async () => ({
      comp_main: '12.5',
      comp_eu: '11.0',
    }));
    const wrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues: vi.fn(), Browse: vi.fn() } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
      draftValue: 'draft-eu',
    });
    await flushOpen();
    const euIdx = inputIndexByLabel(wrapper, 'EU Company');
    expect(wrapper.findAll('.input')[euIdx]!.element).toMatchObject({ value: 'draft-eu' });
  });

  it('skips draft overlay when auth store is unavailable and draftCompanyId is empty', async () => {
    authStoreState.throwOnAccess = true;
    companyResponses.rows = [];
    const GetFieldCompanyValues = vi.fn(async () => ({
      comp_main: '12.5',
      comp_eu: '11.0',
    }));
    const wrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues: vi.fn(), Browse: vi.fn() } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
      draftValue: 'ignored-draft',
    });
    await flushOpen();
    // No allowed/enabled → fallback uses map keys; Search returns [] so labels are ids.
    expect(wrapper.findAll('.input').length).toBeGreaterThan(0);
    expect(wrapper.findAll('.input')[0]!.element).not.toMatchObject({ value: 'ignored-draft' });
  });

  it('clears draft with null, emits closed, and skips blank company ids', async () => {
    authStoreState.throwOnAccess = false;
    authStoreState.metadata = {
      allowedCompanyIds: ['', 'comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_eu',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main Company' },
      { Id: 'comp_eu', DisplayName: 'EU Company' },
    ];
    const GetFieldCompanyValues = vi.fn(async () => ({ comp_main: '12.5', comp_eu: '11.0' }));
    const wrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues: vi.fn(), Browse: vi.fn() } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
      draftCompanyId: 'comp_eu',
      draftValue: null,
    });
    await flushOpen();
    expect(wrapper.findAll('.input')).toHaveLength(2);
    const euIdx = inputIndexByLabel(wrapper, 'EU Company');
    expect(wrapper.findAll('.input')[euIdx]!.element).toMatchObject({ value: '' });
    await wrapper.find('.emit-closed').trigger('click');
    expect(wrapper.emitted('closed')).toBeTruthy();
  });

  it('treats non-object company maps as empty and still saves new values', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main Company' },
      { Id: 'comp_eu', DisplayName: 'EU Company' },
    ];
    const UpdateFieldCompanyValues = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({ Cost: 'Hello' }));
    const GetFieldCompanyValues = vi.fn(async () => null as any);
    const wrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues, Browse } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
    });
    await flushOpen();
    await wrapper.findAll('.input')[0]!.setValue('Hello');
    await wrapper.findAll('.input')[1]!.setValue('你好');
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    const patch = UpdateFieldCompanyValues.mock.calls[0]![2] as Record<string, string>;
    expect(Object.values(patch).sort()).toEqual(['Hello', '你好'].sort());
    expect(wrapper.emitted('saved')?.[0]?.[0]).toBe('Hello');
  });

  it('emits saved null when Browse omits the field', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main'],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [{ Id: 'comp_main', DisplayName: 'Main Company' }];
    const UpdateFieldCompanyValues = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({}));
    const GetFieldCompanyValues = vi.fn(async () => ({ comp_main: 'Hello' }));
    const wrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues, Browse } as any,
      recordId: 'prod-1',
      fieldName: 'Cost',
    });
    await flushOpen();
    await wrapper.findAll('.input')[0]!.setValue('Hello!');
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(wrapper.emitted('saved')?.[0]?.[0]).toBeNull();
  });
});
