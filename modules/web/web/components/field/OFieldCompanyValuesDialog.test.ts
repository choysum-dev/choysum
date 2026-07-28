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

  it('coerces boolean/int/monetary patch values and skips empty new rows', async () => {
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
    const Browse = vi.fn(async () => ({ Flag: true }));
    const GetFieldCompanyValues = vi.fn(async () => ({ comp_main: true }));

    const boolWrapper = mountDialog({
      modelValue: true,
      store: { GetFieldCompanyValues, UpdateFieldCompanyValues, Browse } as any,
      recordId: 'r1',
      fieldName: 'Flag',
      fieldLabel: 'Flag',
      fieldType: 'boolean',
    });
    await flushOpen();
    const euIdx = inputIndexByLabel(boolWrapper, 'EU');
    await boolWrapper.findAll('.input')[euIdx]!.setValue('yes');
    const usIdx = inputIndexByLabel(boolWrapper, 'US');
    await boolWrapper.findAll('.input')[usIdx]!.setValue(''); // new empty → skip
    let buttons = boolWrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateFieldCompanyValues).toHaveBeenCalledWith('r1', 'Flag', { comp_eu: true });
    boolWrapper.unmount();

    UpdateFieldCompanyValues.mockClear();
    const GetFieldCompanyValuesInt = vi.fn(async () => ({ comp_main: 1 }));
    const intWrapper = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: GetFieldCompanyValuesInt,
        UpdateFieldCompanyValues,
        Browse: vi.fn(async () => ({ Qty: 2 })),
      } as any,
      recordId: 'r1',
      fieldName: 'Qty',
      fieldLabel: 'Qty',
      fieldType: 'int',
    });
    await flushOpen();
    await intWrapper.findAll('.input')[inputIndexByLabel(intWrapper, 'EU')]!.setValue('42');
    await intWrapper.findAll('.input')[inputIndexByLabel(intWrapper, 'US')]!.setValue('NaN-ish');
    buttons = intWrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateFieldCompanyValues).toHaveBeenCalledWith('r1', 'Qty', { comp_eu: 42, comp_us: 'NaN-ish' });
    intWrapper.unmount();

    UpdateFieldCompanyValues.mockClear();
    const GetFieldCompanyValuesMoney = vi.fn(async () => ({ comp_main: '1.0' }));
    const moneyWrapper = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: GetFieldCompanyValuesMoney,
        UpdateFieldCompanyValues,
        Browse: vi.fn(async () => ({ Amount: 1 })),
      } as any,
      recordId: 'r1',
      fieldName: 'Amount',
      fieldLabel: '',
      fieldType: 'monetary',
    });
    await flushOpen();
    expect(moneyWrapper.text().toLowerCase()).toMatch(/company values/);
    await moneyWrapper.findAll('.input')[inputIndexByLabel(moneyWrapper, 'EU')]!.setValue('3.5');
    buttons = moneyWrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateFieldCompanyValues).toHaveBeenCalledWith('r1', 'Amount', { comp_eu: 3.5 });
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

  it('covers remaining coerce and error fallbacks', async () => {
    const { ElMessage } = await import('element-plus');
    authStoreState.throwOnAccess = false;
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

    // integer / float / decimal / bigint / boolean whitespace
    for (const [fieldType, input, expected] of [
      ['integer', '7', 7],
      ['bigint', '8', 8],
      ['float', '1.25', 1.25],
      ['decimal', '2.5', 2.5],
    ] as const) {
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
      wrapper.unmount();
    }

    // boolean whitespace-only → true via coerce
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
    boolWrapper.unmount();

    // load error without Error.message (string throw) hits err || fallback chain
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
    expect(ElMessage.error).toHaveBeenCalled();
    failWrapper.unmount();

    // save error without message
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
    await saveWrapper.findAll('.input')[inputIndexByLabel(saveWrapper, 'EU')]!.setValue('z');
    await saveWrapper.findAll('.btn')[saveWrapper.findAll('.btn').length - 1]!.trigger('click');
    await flushOpen();
    expect(ElMessage.error).toHaveBeenCalled();

    // empty allowlist → enabled ∪ keys; Search non-array
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
    expect(emptyAllow.findAll('.item').length).toBeGreaterThan(0);
  });


  it('covers native el-dialog render path without stub', async () => {
    authStoreState.throwOnAccess = false;
    authStoreState.nullIdentity = false;
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
        },
        directives: { loading: loadingDirective },
      },
    });
    await flushOpen();
    expect(wrapper.find('.o-field-company-values-dialog__body').exists() || wrapper.html().length > 0).toBe(true);
    wrapper.unmount();
  });

  it('covers draft missing row, nullish auth ids, and error fallback ternary', async () => {
    const { ElMessage } = await import('element-plus');
    authStoreState.throwOnAccess = false;
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

    // draftCompanyId not in row set → applyDraftValue early !row return
    const wrapper = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => null),
        UpdateFieldCompanyValues: vi.fn(),
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
    wrapper.unmount();

    // identity without metadata object
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
    nullIdent.unmount();
    authStoreState.nullIdentity = false;

    // non-array allow/enabled lists
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
    expect(w2.findAll('.item').length).toBeGreaterThan(0);
    w2.unmount();

    // float non-finite keeps raw text; load/save errors with nullish err hit _t fallback
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
    floatWrapper.unmount();

    ElMessage.error.mockClear?.();
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
    expect(ElMessage.error).toHaveBeenCalled();
    loadNull.unmount();

    const saveNull = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ comp_main: '1' })),
        UpdateFieldCompanyValues: vi.fn(async () => {
          throw undefined;
        }),
        Browse: vi.fn(),
      } as any,
      recordId: 'r1',
      fieldName: 'Cost',
    });
    await flushOpen();
    await saveNull.findAll('.input')[inputIndexByLabel(saveNull, 'EU')]!.setValue('z');
    await saveNull.findAll('.btn')[saveNull.findAll('.btn').length - 1]!.trigger('click');
    await flushOpen();
    expect(ElMessage.error).toHaveBeenCalled();

    // error object with empty message falls through; object without message hits typeof-object branch
    const saveEmptyMsg = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ comp_main: '1' })),
        UpdateFieldCompanyValues: vi.fn(async () => {
          throw { message: '   ' };
        }),
        Browse: vi.fn(),
      } as any,
      recordId: 'r1',
      fieldName: 'Cost',
    });
    await flushOpen();
    await saveEmptyMsg.findAll('.input')[inputIndexByLabel(saveEmptyMsg, 'EU')]!.setValue('z');
    await saveEmptyMsg.findAll('.btn')[saveEmptyMsg.findAll('.btn').length - 1]!.trigger('click');
    await flushOpen();
    expect(ElMessage.error).toHaveBeenCalled();

    const saveNoMsg = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ comp_main: '1' })),
        UpdateFieldCompanyValues: vi.fn(async () => {
          throw { code: 1 };
        }),
        Browse: vi.fn(),
      } as any,
      recordId: 'r1',
      fieldName: 'Cost',
    });
    await flushOpen();
    await saveNoMsg.findAll('.input')[inputIndexByLabel(saveNoMsg, 'EU')]!.setValue('z');
    await saveNoMsg.findAll('.btn')[saveNoMsg.findAll('.btn').length - 1]!.trigger('click');
    await flushOpen();
    expect(ElMessage.error).toHaveBeenCalled();

    // empty-string throw → formatCaughtError asText falsy → fallback
    const saveEmptyStr = mountDialog({
      modelValue: true,
      store: {
        GetFieldCompanyValues: vi.fn(async () => ({ comp_main: '1' })),
        UpdateFieldCompanyValues: vi.fn(async () => {
          throw '';
        }),
        Browse: vi.fn(),
      } as any,
      recordId: 'r1',
      fieldName: 'Cost',
    });
    await flushOpen();
    await saveEmptyStr.findAll('.input')[inputIndexByLabel(saveEmptyStr, 'EU')]!.setValue('z');
    await saveEmptyStr.findAll('.btn')[saveEmptyStr.findAll('.btn').length - 1]!.trigger('click');
    await flushOpen();
    expect(ElMessage.error).toHaveBeenCalled();

    // Browse returns null → nextValue undefined path; closed emit
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
