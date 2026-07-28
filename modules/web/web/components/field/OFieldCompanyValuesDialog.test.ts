// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';

import OFieldCompanyValuesDialog from './OFieldCompanyValuesDialog.vue';

const DEFAULT_COMPANY_ROWS = [
  { Id: 'comp_main', DisplayName: 'Main Company' },
  { Id: 'comp_eu', DisplayName: 'EU Company' },
] as Array<{ Id: string; DisplayName: string }>;

const DEFAULT_AUTH_METADATA = {
  allowedCompanyIds: ['comp_main', 'comp_eu'] as string[],
  enabledCompanyIds: ['comp_main'] as string[],
  activeCompanyId: 'comp_main',
};

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
    } as Record<string, unknown> | null,
    throwOnAccess: false,
    nullIdentity: false,
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
    if (authStoreState.nullIdentity) return { identity: null };
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
    props: ['loading', 'type', 'nativeType'],
    emits: ['click'],
    template: '<button class="btn" type="button" @click="$emit(\'click\')"><slot /></button>',
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
  beforeEach(async () => {
    authStoreState.throwOnAccess = false;
    authStoreState.nullIdentity = false;
    authStoreState.metadata = { ...DEFAULT_AUTH_METADATA };
    companyResponses.rows = DEFAULT_COMPANY_ROWS.map(r => ({ ...r }));
    const { ElMessage } = await import('element-plus');
    vi.mocked(ElMessage.error).mockClear();
    vi.mocked(ElMessage.success).mockClear();
  });

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
      fieldType: 'number',
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

    expect(UpdateFieldCompanyValues).toHaveBeenCalledWith('prod-1', 'Cost', { comp_eu: 11.5 });
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
    expect(labels).toEqual(expect.arrayContaining(['Main Company', 'EU Company']));
    expect(labels).toHaveLength(2);
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

  it.each([
    {
      name: 'boolean yes token',
      fieldType: 'boolean',
      fieldName: 'Flag',
      fieldLabel: 'Flag',
      initial: { comp_main: true },
      browse: { Flag: true },
      edits: [
        { label: 'EU', value: 'yes' },
        { label: 'US', value: '' },
      ],
      expectedPatch: { comp_eu: true },
    },
    {
      name: 'int finite and non-finite',
      fieldType: 'int',
      fieldName: 'Qty',
      fieldLabel: 'Qty',
      initial: { comp_main: 1 },
      browse: { Qty: 2 },
      edits: [
        { label: 'EU', value: '42' },
        { label: 'US', value: 'NaN-ish' },
      ],
      expectedPatch: { comp_eu: 42, comp_us: 'NaN-ish' },
    },
    {
      name: 'monetary number',
      fieldType: 'monetary',
      fieldName: 'Amount',
      fieldLabel: '',
      initial: { comp_main: '1.0' },
      browse: { Amount: 1 },
      edits: [{ label: 'EU', value: '3.5' }],
      expectedPatch: { comp_eu: 3.5 },
    },
  ] as const)('coerces patch values: $name', async ({ fieldType, fieldName, fieldLabel, initial, browse, edits, expectedPatch }) => {
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu', 'comp_us'],
      enabledCompanyIds: ['comp_main', 'comp_eu', 'comp_us'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main' },
      { Id: 'comp_eu', DisplayName: 'EU' },
      { Id: 'comp_us', DisplayName: 'US' },
    ];

    const UpdateFieldCompanyValues = vi.fn(async () => true);
    const wrapper = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ ...initial })),
        UpdateFieldCompanyValues,
        Browse: vi.fn(async () => ({ ...browse })),
      } as any,
      recordId: 'r1',
      fieldName,
      fieldLabel,
      fieldType,
    });
    await flushOpen();
    for (const edit of edits) {
      await wrapper.findAll('.input')[inputIndexByLabel(wrapper, edit.label)]!.setValue(edit.value);
    }
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateFieldCompanyValues).toHaveBeenCalledWith('r1', fieldName, expectedPatch);
  });

  it('coerces boolean falsey tokens and empty boolean text', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main', 'comp_eu'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main' },
      { Id: 'comp_eu', DisplayName: 'EU' },
    ];
    const UpdateFieldCompanyValues = vi.fn(async () => true);
    const wrapper = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ comp_main: true })),
        UpdateFieldCompanyValues,
        Browse: vi.fn(async () => ({ Flag: false })),
      } as any,
      recordId: 'r1',
      fieldName: 'Flag',
      fieldLabel: 'Flag',
      fieldType: 'boolean',
    });
    await flushOpen();
    await wrapper.findAll('.input')[inputIndexByLabel(wrapper, 'EU')]!.setValue('0');
    await wrapper.findAll('.input')[inputIndexByLabel(wrapper, 'Main')]!.setValue('no');
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateFieldCompanyValues).toHaveBeenCalledWith('r1', 'Flag', { comp_eu: false, comp_main: false });
  });


  it('coerces whitespace-only boolean as true', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main', 'comp_eu'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main' },
      { Id: 'comp_eu', DisplayName: 'EU' },
    ];
    const UpdateBool = vi.fn(async () => true);
    const boolWrapper = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ comp_main: false, comp_eu: false })),
        UpdateFieldCompanyValues: UpdateBool,
        Browse: vi.fn(async () => ({ Flag: true })),
      } as any,
      recordId: 'r1',
      fieldName: 'Flag',
      fieldType: 'boolean',
    });
    await flushOpen();
    await boolWrapper.findAll('.input')[inputIndexByLabel(boolWrapper, 'EU')]!.setValue('   ');
    await boolWrapper.findAll('.btn')[boolWrapper.findAll('.btn').length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateBool).toHaveBeenCalledWith('r1', 'Flag', { comp_eu: true });
  });

  it.each([
    ['integer', '7', 7],
    ['bigint', '8', 8],
    ['float', '1.25', 1.25],
    ['decimal', '2.5', 2.5],
  ] as const)('coerces %s patch values', async (fieldType, input, expected) => {
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main', 'comp_eu'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main' },
      { Id: '', DisplayName: 'Ghost' },
      { Id: 'comp_eu', DisplayName: 'EU' },
    ];
    const UpdateFieldCompanyValues = vi.fn(async () => true);
    const wrapper = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ comp_main: '1' })),
        UpdateFieldCompanyValues,
        Browse: vi.fn(async () => ({ X: expected })),
      } as any,
      recordId: 'r1',
      fieldName: 'X',
      fieldLabel: 'X',
      fieldType,
    });
    await flushOpen();
    await wrapper.findAll('.input')[inputIndexByLabel(wrapper, 'EU')]!.setValue(input);
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateFieldCompanyValues).toHaveBeenCalledWith('r1', 'X', { comp_eu: expected });
  });

  it('shows load error for string throw and save error for string throw', async () => {
    const { ElMessage } = await import('element-plus');
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main', 'comp_eu'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main' },
      { Id: 'comp_eu', DisplayName: 'EU' },
    ];
    const GetFail = vi.fn(async () => {
      throw 'load-string';
    });
    const failWrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues: GetFail, UpdateFieldCompanyValues: vi.fn(), Browse: vi.fn() } as any,
      recordId: 'r1',
      fieldName: 'Cost',
    });
    await flushOpen();
    expect(GetFail).toHaveBeenCalled();
    expect(ElMessage.error).toHaveBeenCalledTimes(1);
    expect(ElMessage.error).toHaveBeenCalledWith('load-string');
    failWrapper.unmount();

    vi.mocked(ElMessage.error).mockClear();
    const UpdateFail = vi.fn(async () => {
      throw 'save-string';
    });
    const saveWrapper = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ comp_main: '1' })),
        UpdateFieldCompanyValues: UpdateFail,
        Browse: vi.fn(),
      } as any,
      recordId: 'r1',
      fieldName: 'Cost',
      fieldType: 'char',
    });
    await flushOpen();
    vi.mocked(ElMessage.error).mockClear();
    await saveWrapper.findAll('.input')[inputIndexByLabel(saveWrapper, 'EU')]!.setValue('z');
    await saveWrapper.findAll('.btn')[saveWrapper.findAll('.btn').length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateFail).toHaveBeenCalledTimes(1);
    expect(ElMessage.error).toHaveBeenCalledTimes(1);
    expect(ElMessage.error).toHaveBeenCalledWith('save-string');
  });

  it('falls back to enabled ∪ map keys for empty allowlist with non-array Search', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: [],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = null as any;
    const emptyAllow = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ comp_main: '1', extra: '2' })),
        UpdateFieldCompanyValues: vi.fn(),
        Browse: vi.fn(),
      } as any,
      recordId: 'r1',
      fieldName: 'Cost',
      maxLength: undefined,
    });
    await flushOpen();
    const labels = emptyAllow.findAll('.item').map(el => el.attributes('data-label'));
    expect(labels).toEqual(expect.arrayContaining(['comp_main', 'extra']));
    expect(labels).toHaveLength(2);
  });

  it('covers native el-dialog render path without stub', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main'],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [{ Id: 'comp_main', DisplayName: 'Main' }];
    const wrapper = mount(OFieldCompanyValuesDialog, {
      props: {
        modelValue: true,
        store: {
          GetFieldCompanyValues: vi.fn(async () => ({ comp_main: '1' })),
          UpdateFieldCompanyValues: vi.fn(),
          Browse: vi.fn(),
        } as any,
        recordId: 'r1',
        fieldName: 'Cost',
      } as any,
      global: {
        stubs: {
          ...dialogStubs,
          'el-dialog': false,
          // Keep dialog content in-tree so body is queryable under happy-dom.
          Teleport: true,
        },
        directives: { loading: loadingDirective },
      },
    });
    await flushOpen();
    expect(wrapper.find('.o-field-company-values-dialog__body').exists()).toBe(true);
    wrapper.unmount();
  });

  it('applies draft only when company row exists', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: [null, 'comp_main', '', 'comp_eu', 'comp_main'] as any,
      enabledCompanyIds: [undefined, 'comp_main'] as any,
      activeCompanyId: null,
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: undefined as any },
      { Id: 'comp_eu', DisplayName: 'EU' },
      { Id: '', DisplayName: 'skip' },
    ];
    const UpdateFieldCompanyValues = vi.fn(async () => true);
    const wrapper = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => null),
        UpdateFieldCompanyValues,
        Browse: vi.fn(),
      } as any,
      recordId: 'r1',
      fieldName: '',
      fieldLabel: '',
      draftCompanyId: 'comp_missing',
      draftValue: 'ghost-draft',
      maxLength: 10,
    });
    await flushOpen();
    expect(wrapper.find('.dialog').attributes('data-title')?.toLowerCase()).toMatch(/company values/);
    expect(wrapper.findAll('.input')[0]!.attributes('data-maxlength')).toBe('10');
    expect(wrapper.findAll('.input')[0]!.attributes('data-word-limit')).not.toBe('false');
    await wrapper.find('form').trigger('submit');
    expect(UpdateFieldCompanyValues).not.toHaveBeenCalled();
    expect(wrapper.find('.dialog').exists()).toBe(true);
  });

  it('renders no rows when identity is null', async () => {
    authStoreState.nullIdentity = true;
    companyResponses.rows = [];
    const nullIdent = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({})),
        UpdateFieldCompanyValues: vi.fn(),
        Browse: vi.fn(),
      } as any,
      recordId: 'r1',
      fieldName: 'Cost',
    });
    await flushOpen();
    expect(nullIdent.findAll('.item').length).toBe(0);
  });

  it('falls back to map keys when allow/enabled lists are non-arrays', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: 'nope' as any,
      enabledCompanyIds: null as any,
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [{ Id: 'comp_main', DisplayName: 'Main' }];
    const w2 = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ comp_main: '1', orphan: '2' })),
        UpdateFieldCompanyValues: vi.fn(),
        Browse: vi.fn(),
      } as any,
      recordId: 'r1',
      fieldName: 'Cost',
    });
    await flushOpen();
    const labels = w2.findAll('.item').map(el => el.attributes('data-label'));
    expect(labels).toEqual(expect.arrayContaining(['Main', 'orphan']));
    expect(labels).toHaveLength(2);
  });

  it('keeps non-finite float text in the patch', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main', 'comp_eu'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main' },
      { Id: 'comp_eu', DisplayName: 'EU' },
    ];
    const UpdateFieldCompanyValues = vi.fn(async () => true);
    const floatWrapper = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ comp_main: '1' })),
        UpdateFieldCompanyValues,
        Browse: vi.fn(async () => ({ Amount: 'x' })),
      } as any,
      recordId: 'r1',
      fieldName: 'Amount',
      fieldType: 'float',
    });
    await flushOpen();
    await floatWrapper.findAll('.input')[inputIndexByLabel(floatWrapper, 'EU')]!.setValue('not-a-num');
    await floatWrapper.findAll('.btn')[floatWrapper.findAll('.btn').length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateFieldCompanyValues).toHaveBeenCalledWith('r1', 'Amount', { comp_eu: 'not-a-num' });
  });

  it.each([
    { name: 'null throw', throwValue: null },
    { name: 'undefined throw', throwValue: undefined },
    { name: 'blank message', throwValue: { message: '   ' } },
    { name: 'object without message', throwValue: { code: 1 } },
    { name: 'empty string', throwValue: '' },
  ])('formats save error fallback for $name', async ({ throwValue }) => {
    const { ElMessage } = await import('element-plus');
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main', 'comp_eu'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main' },
      { Id: 'comp_eu', DisplayName: 'EU' },
    ];
    const UpdateFail = vi.fn(async () => {
      throw throwValue;
    });
    const wrapper = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ comp_main: '1' })),
        UpdateFieldCompanyValues: UpdateFail,
        Browse: vi.fn(),
      } as any,
      recordId: 'r1',
      fieldName: 'Cost',
    });
    await flushOpen();
    vi.mocked(ElMessage.error).mockClear();
    await wrapper.findAll('.input')[inputIndexByLabel(wrapper, 'EU')]!.setValue('z');
    await wrapper.findAll('.btn')[wrapper.findAll('.btn').length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateFail).toHaveBeenCalledTimes(1);
    expect(ElMessage.error).toHaveBeenCalledTimes(1);
    expect(String(vi.mocked(ElMessage.error).mock.calls.at(-1)?.[0] ?? '')).toMatch(/Failed to save company values/i);
  });

  it('formats load error fallback for null throw', async () => {
    const { ElMessage } = await import('element-plus');
    vi.mocked(ElMessage.error).mockClear();
    const loadNull = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => {
          throw null;
        }),
        UpdateFieldCompanyValues: vi.fn(),
        Browse: vi.fn(),
      } as any,
      recordId: 'r1',
      fieldName: 'Cost',
    });
    await flushOpen();
    expect(ElMessage.error).toHaveBeenCalledTimes(1);
    loadNull.unmount();
  });

  it('emits null saved value when Browse returns null', async () => {
    authStoreState.metadata = {
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main', 'comp_eu'],
      activeCompanyId: 'comp_main',
    };
    companyResponses.rows = [
      { Id: 'comp_main', DisplayName: 'Main' },
      { Id: 'comp_eu', DisplayName: 'EU' },
    ];
    const browseNull = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ comp_main: '1' })),
        UpdateFieldCompanyValues: vi.fn(async () => true),
        Browse: vi.fn(async () => null),
      } as any,
      recordId: 'r1',
      fieldName: 'Cost',
      fieldType: 'char',
    });
    await flushOpen();
    await browseNull.findAll('.input')[inputIndexByLabel(browseNull, 'EU')]!.setValue('z');
    await browseNull.findAll('.btn')[browseNull.findAll('.btn').length - 1]!.trigger('click');
    await flushOpen();
    expect(browseNull.emitted('saved')?.[0]?.[0]).toBeNull();
    await browseNull.find('.emit-closed').trigger('click');
    expect(browseNull.emitted('closed')).toBeTruthy();
  });
});
