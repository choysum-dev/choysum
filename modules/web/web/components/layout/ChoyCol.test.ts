// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyCol from './ChoyCol.vue';
import ChoyGrid from './ChoyGrid.vue';

function mountInGrid(
  colProps: Record<string, unknown> = {},
  gridProps: Record<string, unknown> = {},
  slot: () => unknown = () => 'cell',
) {
  return mountApp(ChoyGrid as any, {
    props: gridProps,
    slots: {
      default: () =>
        h(ChoyCol as any, { ...colProps, 'data-test': 'col' }, slot),
    },
  });
}

/** Minimal field chrome so Col stack `:has(> [data-slot=field])` engages. */
function fakeField(label: string, orientation: 'horizontal' | 'vertical' = 'horizontal') {
  return h(
    'div',
    {
      'data-slot': 'field',
      'data-orientation': orientation,
      class: 'choy-field-base',
    },
    [
      orientation === 'horizontal'
        ? h('div', { class: 'choy-field-base__label-row' }, label)
        : null,
      h('div', { class: 'choy-field-base__content' }, 'control'),
    ],
  );
}

describe('ChoyCol', () => {
  test('sets 421 CSS vars when span is omitted under a responsive grid', async () => {
    const mounted = mountInGrid();
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement | null;
    expect(col).not.toBeNull();
    expect(col!.style.getPropertyValue('--choy-col-span-base').trim()).toBe('12');
    expect(col!.style.getPropertyValue('--choy-col-span-md').trim()).toBe('6');
    expect(col!.style.getPropertyValue('--choy-col-span-lg').trim()).toBe('3');
    mounted.unmount();
  });

  test('scales numeric span under responsive grid', async () => {
    const mounted = mountInGrid({ span: 3 });
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement;
    expect(col.style.getPropertyValue('--choy-col-span-lg').trim()).toBe('3');
    expect(col.style.getPropertyValue('--choy-col-span-md').trim()).toBe('6');
    expect(col.style.getPropertyValue('--choy-col-span-base').trim()).toBe('12');
    mounted.unmount();
  });

  test('keeps numeric span fixed when grid responsive is false', async () => {
    const mounted = mountInGrid({ span: 3 }, { responsive: false });
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement;
    expect(col.style.getPropertyValue('--choy-col-span-base').trim()).toBe('3');
    expect(col.style.getPropertyValue('--choy-col-span-md').trim()).toBe('3');
    expect(col.style.getPropertyValue('--choy-col-span-lg').trim()).toBe('3');
    mounted.unmount();
  });

  test('honors object span bands', async () => {
    const mounted = mountInGrid({ span: { base: 12, md: 6, lg: 3 } });
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement;
    expect(col.style.getPropertyValue('--choy-col-span-base').trim()).toBe('12');
    expect(col.style.getPropertyValue('--choy-col-span-md').trim()).toBe('6');
    expect(col.style.getPropertyValue('--choy-col-span-lg').trim()).toBe('3');
    mounted.unmount();
  });

  test('defaults to label-gutter host with inner stack for fields', async () => {
    const mounted = mountInGrid({}, {}, () => [
      fakeField('Name'),
      fakeField('Title'),
    ]);
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement;
    expect(col.getAttribute('data-label-gutter')).toBe('true');
    expect(col.querySelector('.choy-col__stack')).not.toBeNull();
    expect(col.querySelectorAll('[data-slot="field"]').length).toBe(2);
    mounted.unmount();
  });

  test('label-gutter=false skips stack host', async () => {
    const mounted = mountInGrid({ labelGutter: false }, {}, () => fakeField('Alone'));
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement;
    expect(col.getAttribute('data-label-gutter')).toBe('false');
    expect(col.querySelector('.choy-col__stack')).toBeNull();
    expect(col.querySelector('[data-slot="field"]')).not.toBeNull();
    mounted.unmount();
  });

  test('label-width writes --choy-field-label-width on the Col', async () => {
    const mounted = mountInGrid(
      { labelWidth: '8rem', span: 6 },
      {},
      () => [fakeField('Short'), fakeField('Longer label text')],
    );
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement;
    expect(col.getAttribute('data-label-width')).toBe('true');
    expect(col.style.getPropertyValue('--choy-field-label-width').trim()).toBe('8rem');
    mounted.unmount();
  });

  test('omits label-width attr and var when prop is empty', async () => {
    const mounted = mountInGrid({ labelWidth: '  ' }, {}, () => fakeField('A'));
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement;
    expect(col.getAttribute('data-label-width')).toBeNull();
    expect(col.style.getPropertyValue('--choy-field-label-width').trim()).toBe('');
    mounted.unmount();
  });

  test('vertical fields still render inside the stack', async () => {
    const mounted = mountInGrid({ span: 12 }, {}, () => [
      fakeField('Notes', 'vertical'),
      fakeField('Title'),
    ]);
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement;
    const stack = col.querySelector('.choy-col__stack');
    expect(stack).not.toBeNull();
    expect(stack!.querySelectorAll('[data-orientation="vertical"]').length).toBe(1);
    expect(stack!.querySelectorAll('[data-orientation="horizontal"]').length).toBe(1);
    mounted.unmount();
  });

  test('single-field Col keeps one field for default 421 slots', async () => {
    const mounted = mountInGrid({}, {}, () => fakeField('Code'));
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement;
    expect(col.querySelectorAll('[data-slot="field"]').length).toBe(1);
    expect(col.getAttribute('data-label-gutter')).toBe('true');
    mounted.unmount();
  });

  test('horizontal fields expose label-row and content for contents stacking', async () => {
    const mounted = mountInGrid({ span: 6 }, {}, () => [
      fakeField('编码'),
      fakeField('字符名称'),
    ]);
    await flushPromises();
    const stack = mounted.q('.choy-col__stack') as HTMLElement;
    expect(stack).not.toBeNull();
    const fields = stack.querySelectorAll('[data-slot="field"]');
    expect(fields.length).toBe(2);
    for (const field of Array.from(fields)) {
      expect(field.getAttribute('data-orientation')).toBe('horizontal');
      expect(field.querySelector('.choy-field-base__label-row')).not.toBeNull();
      expect(field.querySelector('.choy-field-base__content')).not.toBeNull();
    }
    mounted.unmount();
  });
});
