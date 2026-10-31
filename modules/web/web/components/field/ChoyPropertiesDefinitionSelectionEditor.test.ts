// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick } from 'vue';

import { mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyPropertiesDefinitionSelectionEditor from './ChoyPropertiesDefinitionSelectionEditor.vue';
import type { SelectionDraftRow } from './propertiesDefinitionHelpers';

function installChoyButtonStub() {
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    props: {
      type: String,
      size: String,
      variant: String,
      disabled: Boolean,
      class: [String, Array, Object],
    },
    emits: ['click'],
    setup(props: any, { emit, slots, attrs }: any) {
      return () =>
        h(
          'button',
          {
            ...attrs,
            type: 'button',
            disabled: props.disabled || undefined,
            onClick: (e: Event) => emit('click', e),
          },
          slots.default?.(),
        );
    },
  } as any);
}

function mountEditor(initial: SelectionDraftRow[], extra: Record<string, unknown> = {}) {
  let mounted: ReturnType<typeof mountApp>;
  mounted = mountApp(ChoyPropertiesDefinitionSelectionEditor as any, {
    reactiveProps: true,
    props: {
      modelValue: initial,
      ...extra,
      'onUpdate:modelValue': (v: SelectionDraftRow[]) => {
        (mounted.props as any).modelValue = v;
      },
    },
  });
  return mounted;
}

describe('ChoyPropertiesDefinitionSelectionEditor', () => {
  beforeEach(() => {
    installChoyButtonStub();
  });

  afterEach(() => {
    restoreSfc(ChoyButton as any);
  });

  test('renders O2M-like rows and Add a line footer', async () => {
    const mounted = mountEditor([
      { value: 'low', label: '低' },
      { value: 'normal', label: '普通' },
    ]);
    await nextTick();
    expect(mounted.q('[data-testid="choy-properties-definition-selection"]')).toBeTruthy();
    expect(mounted.el.querySelectorAll('[data-testid="choy-properties-definition-selection-row"]').length).toBe(2);
    expect(mounted.q('[data-testid="choy-properties-definition-selection-add"]')).toBeTruthy();
    expect(mounted.q('[data-testid="choy-properties-definition-selection-translate"]')).toBeTruthy();
    mounted.unmount();
  });

  test('Add / Remove emit updated rows', async () => {
    const mounted = mountEditor([{ value: 'a', label: 'A' }]);
    await nextTick();
    mounted.click('[data-testid="choy-properties-definition-selection-add"]');
    await nextTick();
    expect((mounted.props as any).modelValue).toEqual([
      { value: 'a', label: 'A' },
      { value: '', label: '' },
    ]);
    expect(mounted.el.querySelectorAll('[data-testid="choy-properties-definition-selection-row"]').length).toBe(2);

    mounted.click('[data-testid="choy-properties-definition-selection-remove"]');
    await nextTick();
    expect((mounted.props as any).modelValue).toEqual([{ value: '', label: '' }]);
    mounted.unmount();
  });

  test('translate is disabled for blank value and emits value when set', async () => {
    const translated: string[] = [];
    const mounted = mountEditor(
      [
        { value: '', label: '' },
        { value: 'high', label: '高' },
      ],
      { onTranslate: (v: string) => translated.push(v) },
    );
    await nextTick();
    const buttons = mounted.el.querySelectorAll(
      '[data-testid="choy-properties-definition-selection-translate"]',
    ) as NodeListOf<HTMLButtonElement>;
    expect(buttons.length).toBe(2);
    expect(buttons[0]?.disabled).toBe(true);
    expect(buttons[1]?.disabled).toBe(false);
    buttons[1]?.click();
    await nextTick();
    expect(translated).toEqual(['high']);
    mounted.unmount();
  });

  test('readonly hides add/remove/translate controls and uses display text', async () => {
    const mounted = mountEditor([{ value: 'x', label: 'X' }], { readonly: true });
    await nextTick();
    expect(mounted.q('[data-testid="choy-properties-definition-selection-add"]')).toBeFalsy();
    expect(mounted.q('[data-testid="choy-properties-definition-selection-remove"]')).toBeFalsy();
    expect(mounted.q('[data-testid="choy-properties-definition-selection-translate"]')).toBeFalsy();
    expect(mounted.q('[data-testid="choy-properties-definition-selection-handle"]')).toBeFalsy();
    expect(mounted.q('input')).toBeFalsy();
    expect(mounted.q('[data-testid="choy-properties-definition-selection-value"]')?.textContent).toBe('x');
    expect(mounted.q('[data-testid="choy-properties-definition-selection-label"]')?.textContent).toBe('X');
    mounted.unmount();
  });

  test('handle drag reorders option rows', async () => {
    const mounted = mountEditor([
      { value: 'low', label: '低' },
      { value: 'normal', label: '普通' },
      { value: 'high', label: '高' },
    ]);
    await nextTick();
    const api = (mounted.root as any)?.handleReorder;
    expect(api).toBeTruthy();
    const dt = {
      effectAllowed: '',
      dropEffect: '',
      setData: () => undefined,
    };
    api.onDragStart(0, { preventDefault() {}, dataTransfer: dt } as unknown as DragEvent);
    await api.onDrop(2, { preventDefault() {}, dataTransfer: dt } as unknown as DragEvent);
    await nextTick();
    expect((mounted.props as any).modelValue.map((r: SelectionDraftRow) => r.value)).toEqual([
      'normal',
      'high',
      'low',
    ]);
    mounted.unmount();
  });

  test('selectionRowDragClass and row drag helpers', async () => {
    const mounted = mountEditor([
      { value: 'a', label: 'A' },
      { value: 'b', label: 'B' },
    ]);
    await nextTick();
    const state = mounted.setupState() as {
      selectionRowDragClass?: (index: number) => Record<string, boolean>;
      onHandleDragStart?: (index: number, event: DragEvent) => void;
      onRowDragOver?: (index: number, event: DragEvent) => void;
      onRowDrop?: (index: number, event: DragEvent) => void;
      handleReorder?: {
        draggingIndex: { value: number | null };
        overIndex: { value: number | null };
      };
    };
    expect(state.selectionRowDragClass?.(0)).toEqual({
      'opacity-60': false,
      'bg-accent/40 ring-1 ring-inset ring-primary/30': false,
    });
    state.handleReorder!.draggingIndex.value = 0;
    state.handleReorder!.overIndex.value = 1;
    expect(state.selectionRowDragClass?.(0)?.['opacity-60']).toBe(true);
    expect(state.selectionRowDragClass?.(1)?.['bg-accent/40 ring-1 ring-inset ring-primary/30']).toBe(true);

    const preventDefault = () => undefined;
    const stopPropagation = () => undefined;
    const setData = () => undefined;
    const setDragImage = () => undefined;
    state.onHandleDragStart?.(0, {
      preventDefault,
      stopPropagation,
      dataTransfer: { effectAllowed: '', setData, setDragImage },
    } as unknown as DragEvent);
    state.onRowDragOver?.(1, { preventDefault } as unknown as DragEvent);
    state.onRowDrop?.(1, { preventDefault } as unknown as DragEvent);
    await nextTick();
    mounted.unmount();

    const readonly = mountEditor([{ value: 'x', label: 'X' }], { readonly: true });
    await nextTick();
    const ro = readonly.setupState() as {
      selectionRowDragClass?: (index: number) => Record<string, boolean>;
    };
    expect(ro.selectionRowDragClass?.(0)).toEqual({});
    readonly.unmount();
  });

  test('null modelValue renders empty state', async () => {
    const mounted = mountEditor(null as any);
    await nextTick();
    expect(mounted.q('[data-testid="choy-properties-definition-selection-empty"]')).toBeTruthy();
    mounted.unmount();
  });

  test('typing value/label inputs emits updated rows', async () => {
    const mounted = mountEditor([{ value: 'a', label: 'A' }]);
    await nextTick();
    const valueInput = mounted.q(
      '[data-testid="choy-properties-definition-selection-value"]',
    ) as HTMLInputElement;
    const labelInput = mounted.q(
      '[data-testid="choy-properties-definition-selection-label"]',
    ) as HTMLInputElement;
    expect(valueInput).toBeTruthy();
    expect(labelInput).toBeTruthy();
    valueInput.value = 'ab';
    valueInput.dispatchEvent(new Event('input', { bubbles: true }));
    await nextTick();
    labelInput.value = 'AB';
    labelInput.dispatchEvent(new Event('input', { bubbles: true }));
    await nextTick();
    expect((mounted.props as any).modelValue[0].value).toBe('ab');
    expect((mounted.props as any).modelValue[0].label).toBe('AB');

    // DOM dragover/drop on the row
    const row = mounted.q('[data-testid="choy-properties-definition-selection-row"]') as HTMLElement;
    const handle = mounted.q(
      '[data-testid="choy-properties-definition-selection-handle"]',
    ) as HTMLElement;
    const dt = {
      effectAllowed: '',
      setData: () => undefined,
      setDragImage: () => undefined,
    };
    handle.dispatchEvent(
      Object.assign(new Event('dragstart', { bubbles: true }), {
        dataTransfer: dt,
        stopPropagation() {},
      }),
    );
    row.dispatchEvent(
      Object.assign(new Event('dragover', { bubbles: true }), {
        preventDefault() {},
        dataTransfer: dt,
      }),
    );
    row.dispatchEvent(
      Object.assign(new Event('drop', { bubbles: true }), {
        preventDefault() {},
        dataTransfer: dt,
      }),
    );
    await nextTick();
    mounted.unmount();
  });
});
