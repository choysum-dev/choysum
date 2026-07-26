// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { computed, nextTick, ref, h } from 'vue';
import { describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import type { UseField } from '@/web/web/composables/useField';
import { useField } from '@/web/web/composables/useField';
import { createFieldsGetHelpers } from '@/web/web/stores/fieldsGet';
import type { WebFieldMetadata } from '@/web/web/stores/modelStore';
import OStatusbarField from './OStatusbarField.vue';

vi.mock('@/web/web/composables/useField', async importOriginal => {
  const mod = await importOriginal<typeof import('@/web/web/composables/useField')>();
  return {
    ...mod,
    useField: vi.fn(mod.useField),
  };
});

function makeBinding(opts: {
  prop?: string;
  isEditMode?: boolean;
  meta?: WebFieldMetadata;
  store?: any;
  value?: string | null;
}): { binding: UseField; value: ReturnType<typeof ref<string | null>> } {
  const value = ref<string | null>(opts.value !== undefined ? opts.value : 'draft');
  const record = ref({ Id: '1', State: value.value });
  const binding = {
    env: {
      isForm: true,
      isEditMode: opts.isEditMode !== false,
      viewMode: opts.isEditMode === false ? 'display' : 'edit',
      fieldPrefix: null,
    },
    prop: opts.prop || 'State',
    meta: opts.meta as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => ({ ...record.value, State: value.value })) as any,
    registerFields: () => {},
    store: opts.store,
    asView: () => ({ fieldValue: () => value }) as any,
  } as UseField;
  return { binding, value };
}

const staticMeta: WebFieldMetadata = {
  id: '1',
  type: 'selection',
  typeAnnotation: 'string',
  string: 'State',
  selection: [
    { value: 'draft', label: 'Draft' },
    { value: 'confirmed', label: 'Confirmed' },
    { value: 'done', label: 'Done' },
  ],
};

function makeStore(meta: WebFieldMetadata = staticMeta) {
  const FieldsGet = vi.fn(async () => ({ State: meta }));
  const helpers = createFieldsGetHelpers({ fieldsMetadata: { State: meta }, FieldsGet }, { getLang: () => 'en_US' });
  return { fieldsMetadata: { State: meta }, FieldsGet, ...helpers };
}

function mountStatusbar(
  props: Record<string, unknown>,
  binding: UseField,
  opts?: {
    slot?: 'edit' | 'display' | 'both';
    provideOnchange?: any;
    captureRules?: { current: any[] | null };
    recordFactory?: () => any;
  }
) {
  const fieldRef = binding.fieldRef();
  const slotMode = opts?.slot ?? 'edit';
  const captured = opts?.captureRules;

  return mount(OStatusbarField as any, {
    props: {
      binding,
      renderMode: 'inline',
      ...props,
    },
    global: {
      provide: opts?.provideOnchange !== undefined ? { lastOnchangeResult: opts.provideOnchange } : {},
      stubs: {
        OFieldBase: {
          props: ['binding', 'readonly', 'renderMode', 'rules', 'formItemProps', 'toView', 'fromView', 'label'],
          setup(p: any, { slots }: any) {
            if (captured) captured.current = p.rules || [];
            if (typeof p.toView === 'function') {
              p.toView('done');
              p.toView(null);
            }
            if (typeof p.fromView === 'function') {
              p.fromView('done');
              p.fromView(null);
            }
            const fieldValue = () => fieldRef as any;
            const record = opts?.recordFactory || (() => ({ Id: '1', State: (fieldRef as any).value }));
            return () => {
              const children: any[] = [];
              if (slotMode === 'edit' || slotMode === 'both') {
                children.push(slots.edit?.({ fieldValue, record }));
              }
              if (slotMode === 'display' || slotMode === 'both') {
                children.push(slots.display?.({ fieldValue, record }));
              }
              return h('div', { class: 'ob', 'data-form-item-class': p.formItemProps?.class }, children);
            };
          },
        },
        'el-segmented': {
          props: ['modelValue', 'options', 'disabled'],
          emits: ['update:modelValue'],
          template: `
            <div
              class="o-statusbar"
              :data-disabled="String(disabled)"
              :data-value="String(modelValue ?? '')"
              :data-options="JSON.stringify(options || [])"
            >
              <button
                v-for="opt in options || []"
                :key="String(opt.value)"
                class="seg-opt"
                :data-value="opt.value"
                :disabled="!!(disabled || opt.disabled)"
                @click="$emit('update:modelValue', opt.value)"
              >{{ opt.label }}</button>
              <button class="seg-empty" @click="$emit('update:modelValue', null)">empty</button>
            </div>
          `,
        },
      },
    },
  });
}

describe('OStatusbarField', () => {
  it('uses chevron class and hides EP selected slider via CSS (D11 contract)', () => {
    const src = readFileSync(resolve(__dirname, './OStatusbarField.vue'), 'utf8');
    expect(src).toContain('class="o-statusbar"');
    expect(src).toContain('el-segmented__item-selected');
    expect(src).toContain('clip-path');
    expect(src).toContain('beforeChange');
    expect(src).not.toMatch(/ElMessageBox/);
  });

  it('OFormView exposes optional #statusbar slot (D6)', () => {
    const src = readFileSync(resolve(__dirname, '../view/OFormView.vue'), 'utf8');
    expect(src).toContain('name="statusbar"');
    expect(src).toMatch(/statusbar\(\):\s*any/);
  });

  it('defaults to non-clickable (disabled segmented)', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store });
    const wrapper = mountStatusbar({}, binding);
    await flushPromises();
    expect(wrapper.get('.o-statusbar').attributes('data-disabled')).toBe('true');
    expect(JSON.parse(wrapper.get('.o-statusbar').attributes('data-options') || '[]').map((o: any) => o.value)).toEqual([
      'draft',
      'confirmed',
      'done',
    ]);
  });

  it('renders both edit and display slots', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store });
    const wrapper = mountStatusbar({ clickable: true }, binding, { slot: 'both' });
    await flushPromises();
    expect(wrapper.findAll('.o-statusbar').length).toBe(2);
  });

  it('clickable writes value; beforeChange false cancels write', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const beforeChange = vi.fn(() => false);
    const wrapper = mountStatusbar({ clickable: true, beforeChange }, binding);
    await flushPromises();

    expect(wrapper.get('.o-statusbar').attributes('data-disabled')).toBe('false');
    await wrapper.get('[data-value="done"]').trigger('click');
    await flushPromises();
    await nextTick();
    expect(beforeChange).toHaveBeenCalledWith('done', 'draft');
    expect(value.value).toBe('draft');

    beforeChange.mockImplementation(() => true);
    await wrapper.get('[data-value="done"]').trigger('click');
    await flushPromises();
    expect(value.value).toBe('done');
  });

  it('writes immediately when clickable and beforeChange omitted', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const wrapper = mountStatusbar({ clickable: true }, binding);
    await flushPromises();
    await wrapper.get('[data-value="confirmed"]').trigger('click');
    await flushPromises();
    expect(value.value).toBe('confirmed');
  });

  it('skips same-value and empty emits; ignores disabled options via onchange', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const onchange = ref({
      selection: [{ field: 'State', selection: ['draft', 'confirmed', 'done'], disabled: ['done'] }],
    });
    const wrapper = mountStatusbar({ clickable: true }, binding, { provideOnchange: onchange });
    await flushPromises();
    await wrapper.get('[data-value="draft"]').trigger('click');
    await flushPromises();
    expect(value.value).toBe('draft');
    await wrapper.get('.seg-empty').trigger('click');
    await flushPromises();
    expect(value.value).toBe('draft');
    await wrapper.get('[data-value="done"]').trigger('click');
    await flushPromises();
    expect(value.value).toBe('draft');
    await wrapper.get('[data-value="confirmed"]').trigger('click');
    await flushPromises();
    expect(value.value).toBe('confirmed');
  });

  it('disables while async beforeChange is pending', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    let release!: (v: boolean) => void;
    const gate = new Promise<boolean>(r => {
      release = r;
    });
    const beforeChange = vi.fn(() => gate);
    const wrapper = mountStatusbar({ clickable: true, beforeChange }, binding);
    await flushPromises();
    const click = wrapper.get('[data-value="done"]').trigger('click');
    await nextTick();
    expect(wrapper.get('.o-statusbar').attributes('data-disabled')).toBe('true');
    // Second click while pending should be ignored.
    await wrapper.get('[data-value="confirmed"]').trigger('click');
    release(true);
    await click;
    await flushPromises();
    expect(value.value).toBe('done');
    expect(beforeChange).toHaveBeenCalledTimes(1);
  });

  it('clickable still writes from display slot (viewMode-independent)', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft', isEditMode: false });
    const wrapper = mountStatusbar({ clickable: true }, binding, { slot: 'display' });
    await flushPromises();
    expect(wrapper.get('.o-statusbar').attributes('data-disabled')).toBe('false');
    await wrapper.get('[data-value="done"]').trigger('click');
    await flushPromises();
    expect(value.value).toBe('done');
  });

  it('respects disabled / readonly boolean / readonly predicate / meta isReadonly', async () => {
    const readonlyMeta = { ...staticMeta, isReadonly: true } as WebFieldMetadata;
    const store = makeStore(readonlyMeta);
    let wrapper = mountStatusbar({ clickable: true }, makeBinding({ meta: readonlyMeta, store }).binding);
    await flushPromises();
    expect(wrapper.get('.o-statusbar').attributes('data-disabled')).toBe('true');
    wrapper.unmount();

    const store2 = makeStore();
    wrapper = mountStatusbar({ clickable: true, disabled: true }, makeBinding({ meta: staticMeta, store: store2 }).binding);
    await flushPromises();
    expect(wrapper.get('.o-statusbar').attributes('data-disabled')).toBe('true');
    wrapper.unmount();

    wrapper = mountStatusbar({ clickable: true, readonly: true }, makeBinding({ meta: staticMeta, store: store2 }).binding);
    await flushPromises();
    expect(wrapper.get('.o-statusbar').attributes('data-disabled')).toBe('true');
    wrapper.unmount();

    wrapper = mountStatusbar(
      { clickable: true, readonly: () => true },
      makeBinding({ meta: staticMeta, store: store2 }).binding
    );
    await flushPromises();
    expect(wrapper.get('.o-statusbar').attributes('data-disabled')).toBe('true');
  });

  it('fails closed when readonly predicate throws', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const wrapper = mountStatusbar(
      {
        clickable: true,
        readonly: () => {
          throw new Error('boom');
        },
      },
      binding
    );
    await flushPromises();
    expect(wrapper.get('.o-statusbar').attributes('data-disabled')).toBe('true');
  });

  it('applies statusbarVisible and selection whitelist; keeps current fallback', async () => {
    const store = makeStore();
    let wrapper = mountStatusbar(
      { statusbarVisible: ['draft', 'done'] },
      makeBinding({ meta: staticMeta, store, value: 'confirmed' }).binding
    );
    await flushPromises();
    expect(JSON.parse(wrapper.get('.o-statusbar').attributes('data-options') || '[]').map((o: any) => o.value)).toEqual([
      'draft',
      'done',
      'confirmed',
    ]);
    wrapper.unmount();

    wrapper = mountStatusbar({ selection: ['done', 'draft'] }, makeBinding({ meta: staticMeta, store, value: 'draft' }).binding);
    await flushPromises();
    expect(JSON.parse(wrapper.get('.o-statusbar').attributes('data-options') || '[]').map((o: any) => o.value)).toEqual([
      'done',
      'draft',
    ]);
  });

  it('falls back to binding.meta when store has no selection; nested prop leaf', async () => {
    const store = {
      fieldsMetadata: {},
      ensureFieldsGet: vi.fn(async () => ({})),
      getFieldMeta: () => undefined,
    };
    const { binding } = makeBinding({
      meta: staticMeta,
      store,
      prop: 'Line.State',
      value: 'draft',
    });
    const wrapper = mountStatusbar({ clickable: true }, binding);
    await flushPromises();
    expect(JSON.parse(wrapper.get('.o-statusbar').attributes('data-options') || '[]').map((o: any) => o.value)).toEqual([
      'draft',
      'confirmed',
      'done',
    ]);
    expect(store.ensureFieldsGet).toHaveBeenCalled();
    expect(store.ensureFieldsGet.mock.calls[0]![0]).toEqual(['State']);
  });

  it('skips ensureFieldsGet when store lacks helper', async () => {
    const store = { fieldsMetadata: { State: staticMeta }, getFieldMeta: () => staticMeta };
    const { binding } = makeBinding({ meta: staticMeta, store });
    mountStatusbar({}, binding);
    await flushPromises();
  });

  it('uses props.store when binding.store is missing; empty prop skips ensure', async () => {
    const store = makeStore();
    const ensureSpy = vi.spyOn(store, 'ensureFieldsGet');
    const { binding } = makeBinding({ meta: staticMeta, store: undefined });
    (binding as any).store = undefined;
    (binding as any).prop = '';
    mountStatusbar({ store }, binding);
    await flushPromises();
    // leafKey is empty → onMounted returns before ensureFieldsGet
    expect(ensureSpy).not.toHaveBeenCalled();
  });

  it('resolves modelStore from props.store when binding.store is absent', async () => {
    const store = makeStore();
    const ensureSpy = vi.spyOn(store, 'ensureFieldsGet');
    const { binding } = makeBinding({ meta: staticMeta, store: undefined });
    (binding as any).store = undefined;
    mountStatusbar({ store }, binding);
    await flushPromises();
    expect(ensureSpy).toHaveBeenCalled();
    expect(ensureSpy.mock.calls[0]![0]).toEqual(['State']);
  });

  it('bootstraps binding via useField when binding prop is omitted', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    vi.mocked(useField).mockReturnValueOnce(binding as any);
    const wrapper = mountStatusbar({ store, prop: 'State', clickable: true }, binding);
    // Remount without binding to hit useField path.
    wrapper.unmount();
    const wrapper2 = mount(OStatusbarField as any, {
      props: { store, prop: 'State', renderMode: 'inline', clickable: true },
      global: {
        stubs: {
          OFieldBase: {
            setup(_: any, { slots }: any) {
              const fieldValue = () => binding.fieldRef() as any;
              const record = () => ({ Id: '1', State: value.value });
              return () => h('div', slots.edit?.({ fieldValue, record }));
            },
          },
          'el-segmented': {
            props: ['modelValue', 'options', 'disabled'],
            emits: ['update:modelValue'],
            template: `
              <div class="o-statusbar" :data-disabled="String(disabled)">
                <button data-value="done" @click="$emit('update:modelValue', 'done')">Done</button>
              </div>
            `,
          },
        },
      },
    });
    await flushPromises();
    expect(useField).toHaveBeenCalled();
    await wrapper2.get('[data-value="done"]').trigger('click');
    await flushPromises();
    expect(value.value).toBe('done');
    wrapper2.unmount();
  });

  it('writes from null current value', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: null });
    const wrapper = mountStatusbar({ clickable: true }, binding);
    await flushPromises();
    await wrapper.get('[data-value="draft"]').trigger('click');
    await flushPromises();
    expect(value.value).toBe('draft');
  });

  it('maps selection entries missing labels via value fallback', async () => {
    const meta = {
      ...staticMeta,
      selection: [{ value: 'draft' }, { value: 'done', label: 'Done' }],
    } as WebFieldMetadata;
    const store = makeStore(meta);
    const { binding } = makeBinding({ meta, store });
    const wrapper = mountStatusbar({}, binding);
    await flushPromises();
    const opts = JSON.parse(wrapper.get('.o-statusbar').attributes('data-options') || '[]');
    expect(opts[0]).toEqual({ label: 'draft', value: 'draft', disabled: false });
  });

  it('handles empty meta selection list', async () => {
    const emptyMeta = { ...staticMeta, selection: [] };
    const store = makeStore(emptyMeta);
    const { binding } = makeBinding({ meta: emptyMeta, store, value: 'x' });
    const wrapper = mountStatusbar({ statusbarVisible: ['a'] }, binding);
    await flushPromises();
    expect(JSON.parse(wrapper.get('.o-statusbar').attributes('data-options') || '[]').map((o: any) => o.value)).toEqual([
      'a',
      'x',
    ]);
  });

  it('merges formItemProps and runs internal validation rules', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const captureRules = { current: null as any[] | null };
    const wrapper = mountStatusbar(
      { formItemProps: { class: 'extra-class' }, rules: [{ required: true, message: 'req' } as any] },
      binding,
      { captureRules }
    );
    await flushPromises();
    expect(String(wrapper.get('.ob').attributes('data-form-item-class'))).toContain('o-statusbar-form-item');
    expect(String(wrapper.get('.ob').attributes('data-form-item-class'))).toContain('extra-class');
    const rules = captureRules.current || [];
    expect(rules.length).toBeGreaterThanOrEqual(2);
    const validator = rules[rules.length - 1]?.validator as Function;
    const cb = vi.fn();
    validator({}, null, cb);
    validator({}, '', cb);
    validator({}, 1, cb);
    validator({}, 'nope', cb);
    validator({}, 'draft', cb);
    expect(cb).toHaveBeenCalled();
    const errors = cb.mock.calls.map((c: any[]) => c[0]).filter(Boolean);
    expect(errors.some((e: Error) => /string|Invalid/i.test(String(e?.message || e)))).toBe(true);
  });

  it('falls back to field value when row omits leaf', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store, value: 'confirmed' });
    const wrapper = mountStatusbar({ clickable: true }, binding, {
      recordFactory: () => ({ Id: '1' }),
    });
    await flushPromises();
    expect(wrapper.get('.o-statusbar').attributes('data-value')).toBe('confirmed');
  });

  it('tolerates fieldRef throw when resolving current from field', async () => {
    const store = makeStore();
    const value = ref<string | null>('draft');
    const binding = {
      env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
      prop: 'State',
      meta: staticMeta,
      fieldRef: () => {
        throw new Error('fieldRef boom');
      },
      fieldRefOf: () => value as any,
      recordRef: () => computed(() => ({ Id: '1' })) as any,
      registerFields: () => {},
      store,
      asView: () => ({ fieldValue: () => value }) as any,
    } as UseField;
    // Stub fieldValue uses a stable ref so the control can still render.
    const wrapper = mount(OStatusbarField as any, {
      props: { binding, renderMode: 'inline', clickable: true },
      global: {
        stubs: {
          OFieldBase: {
            setup(_: any, { slots }: any) {
              const fieldValue = () => value as any;
              const record = () => ({ Id: '1' });
              return () => h('div', slots.edit?.({ fieldValue, record }));
            },
          },
          'el-segmented': {
            props: ['modelValue', 'options', 'disabled'],
            emits: ['update:modelValue'],
            template: `<div class="o-statusbar" :data-options="JSON.stringify(options || [])" />`,
          },
        },
      },
    });
    await flushPromises();
    expect(wrapper.get('.o-statusbar').attributes('data-options')).toBeTruthy();
  });

  it('ensures FieldsGet on mount', async () => {
    const store = makeStore();
    const ensureSpy = vi.spyOn(store, 'ensureFieldsGet');
    const { binding } = makeBinding({ meta: staticMeta, store });
    mountStatusbar({}, binding);
    await flushPromises();
    expect(ensureSpy).toHaveBeenCalled();
    expect(ensureSpy.mock.calls[0]![0]).toEqual(['State']);
  });
});
