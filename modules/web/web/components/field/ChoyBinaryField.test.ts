// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, h, ref } from 'vue';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import BinaryField from './ChoyBinaryField.vue';
import ChoyFieldBase from './ChoyFieldBase.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';

function makeBinding(opts?: { value?: unknown; prop?: string }): any {
  const value = ref(opts?.value ?? null);
  const record = ref({ Id: '1' });
  return {
    env: {
      isForm: true,
      isEditMode: true,
      viewMode: 'edit',
      fieldPrefix: null,
    },
    prop: opts?.prop || 'Doc',
    meta: {} as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => record.value) as any,
    registerFields: () => {},
    store: undefined,
    asView: () => ({ fieldValue: () => value }) as any,
  } as any;
}

function installFieldBaseEditStub() {
  stubSfc(ChoyFieldBase as any, {
    name: 'ChoyFieldBase',
    props: {
      binding: { type: Object, required: false },
      rules: { type: Array, default: () => [] },
    },
    setup(props: any, { slots }: any) {
      return () =>
        h(
          'div',
          { class: 'field-base-stub' },
          slots.edit?.({
            fieldValue: () => (props.binding as UseField | undefined)?.fieldRef?.() ?? ref(null),
            onFieldChange: async () => {},
          })
        );
    },
  });
}

function installChoyButtonStub() {
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    props: {
      class: { type: [String, Object, Array], default: undefined },
      disabled: Boolean,
      size: String,
      variant: String,
      type: String,
      as: String,
    },
    setup(p: any, { attrs, slots }: any) {
      return () =>
        h(
          'button',
          {
            type: 'button',
            class: p.class,
            disabled: p.disabled,
            ...attrs,
          },
          slots.default?.()
        );
    },
  });
}

function installFieldBaseDisplayStub() {
  stubSfc(ChoyFieldBase as any, {
    name: 'ChoyFieldBase',
    props: { binding: { type: Object, required: false } },
    setup(props: any, { slots }: any) {
      return () =>
        h(
          'div',
          slots.display?.({
            fieldValue: () => (props.binding as UseField | undefined)?.fieldRef?.() ?? ref(null),
            renderMode: 'form',
          })
        );
    },
  });
}

function mountField(binding: UseField, mode: 'edit' | 'display' = 'edit', extraProps: Record<string, unknown> = {}) {
  if (mode === 'display') installFieldBaseDisplayStub();
  else installFieldBaseEditStub();
  installChoyButtonStub();
  return mountApp(BinaryField as any, {
    props: {
      binding,
      renderMode: mode === 'display' ? 'display' : 'form',
      drag: false,
      ...extraProps,
    },
  });
}

async function pickFile(m: ReturnType<typeof mountApp>, file: File) {
  const input = m.q('input.sr-only') as HTMLInputElement | null;
  expect(input).toBeTruthy();
  Object.defineProperty(input, 'files', {
    configurable: true,
    value: {
      0: file,
      length: 1,
      item: (i: number) => (i === 0 ? file : null),
    },
  });
  input!.dispatchEvent(new Event('change', { bubbles: true }));
  await flushPromises();
}

describe('ChoyBinaryField', () => {
  afterEach(() => {
    restoreSfc(ChoyFieldBase as any);
    restoreSfc(ChoyButton as any);
  });

  test('renders attachment metadata and chrome', async () => {
    const binding = makeBinding({
      value: {
        attachmentBindingId: '  bind-1  ',
        attachmentObjectId: '  obj-1  ',
        fileName: '  report.pdf  ',
        mimeType: '  application/pdf  ',
        sizeBytes: 2048,
        downloadUrl: '  /files/report.pdf  ',
        kind: 'set',
      },
    });
    const m = mountField(binding);
    await flushPromises();
    expect(m.text()).toContain('report.pdf');
    expect(m.q('.choy-binary-current')).toBeTruthy();
    expect(m.q('.choy-attachment-current--compact')).toBeTruthy();
    expect(m.text()).toMatch(/Download|下载/);
    expect(m.text()).toMatch(/KB/);
    expect(m.text()).not.toMatch(/application\/pdf/);
    expect(m.q('[data-slot=attachment]')).not.toBeNull();
    m.unmount();
  });

  test('display compact row is borderless and keeps mime in tooltip', async () => {
    const m = mountField(
      makeBinding({
        value: {
          fileName: 'report.pdf',
          mimeType: 'application/pdf',
          sizeBytes: 4096,
          downloadUrl: '/files/report.pdf',
          kind: 'set',
        },
      }),
      'display'
    );
    await flushPromises();
    const card = m.q('.choy-attachment-display-card--compact') as HTMLElement | null;
    expect(card).toBeTruthy();
    expect(card?.className).toMatch(/border-transparent/);
    expect(m.q('a.choy-attachment-display-card')?.getAttribute('title')).toMatch(/application\/pdf/);
    expect(m.text()).toContain('report.pdf');
    expect(m.text()).not.toMatch(/application\/pdf/);
    const downloadIcon = Array.from(m.el.querySelectorAll('svg')).find(
      el => el.getAttribute('data-lucide') === 'Download'
    );
    expect(downloadIcon).toBeTruthy();
    m.unmount();
  });

  test('uses mime-specific lucide icons for binary media', async () => {
    const m = mountField(
      makeBinding({
        value: { fileName: 'photo.png', mimeType: 'image/png', kind: 'set' },
      })
    );
    await flushPromises();
    const ss = m.setupState() as any;
    expect(ss.binaryMediaIcon({ mimeType: 'image/png', kind: 'set' }).name).toBe('Image');
    expect(ss.binaryMediaIcon({ mimeType: 'application/pdf', fileName: 'a.pdf', kind: 'set' }).name).toBe('FileText');
    expect(ss.binaryMediaIcon({ fileName: 'a.xlsx', kind: 'set' }).name).toBe('FileSpreadsheet');
    expect(ss.binaryMediaIcon({ fileName: 'a.zip', kind: 'set' }).name).toBe('FileArchive');
    expect(m.q('[data-binary-icon="image"]')).toBeTruthy();
    const svg = Array.from(m.el.querySelectorAll('svg')).find(el => el.getAttribute('data-lucide') === 'Image');
    expect(svg).toBeTruthy();
    m.unmount();
  });

  test('replace is ghost primary; remove is destructive text', async () => {
    const m = mountField(
      makeBinding({
        value: { fileName: 'a.bin', kind: 'set', downloadUrl: '/a.bin' },
      })
    );
    await flushPromises();
    const buttons = Array.from(m.el.querySelectorAll('button')) as HTMLButtonElement[];
    const replace = buttons.find(b => /Replace|替换/.test(b.textContent || ''));
    const remove = buttons.find(b => /Remove|移除/.test(b.textContent || ''));
    expect(replace?.className).toMatch(/text-primary/);
    expect(replace?.className).not.toMatch(/text-destructive/);
    expect(remove?.className).toMatch(/text-destructive/);
    m.unmount();
  });

  test('covers objectId-only and downloadUrl-only attachment resolution', async () => {
    const objectMount = mountField(makeBinding({ value: { attachmentObjectId: '  obj-only  ', kind: 'set' } }));
    expect(objectMount.q('.choy-binary-current')).toBeTruthy();
    objectMount.unmount();
    restoreSfc(ChoyFieldBase as any);

    const downloadMount = mountField(makeBinding({ value: { downloadUrl: '  /files/only.bin  ', kind: 'set' } }), 'display');
    await flushPromises();
    expect(downloadMount.q('a')?.getAttribute('href')).toBe('/files/only.bin');
    downloadMount.unmount();
  });

  test('treats string values and clear/noop kinds', async () => {
    const stringMount = mountField(makeBinding({ value: '  plain-name.bin  ' }));
    expect(stringMount.q('.choy-binary-current')).toBeTruthy();
    stringMount.unmount();
    restoreSfc(ChoyFieldBase as any);

    const clearMount = mountField(makeBinding({ value: { kind: 'clear' } }));
    expect(clearMount.q('.choy-binary-current')).toBeFalsy();
    clearMount.unmount();
  });

  test('empty display uses choy-field-display-empty', async () => {
    const m = mountField(makeBinding({ value: null }), 'display');
    await flushPromises();
    expect(m.q('.choy-field-display-empty')).toBeTruthy();
    m.unmount();
  });

  test('writes pending set envelope on upload', async () => {
    const binding = makeBinding();
    const m = mountField(binding);
    await pickFile(m, new File([new Uint8Array([1, 2])], 'note.txt', { type: 'text/plain' }));
    expect(binding.fieldRef().value).toMatchObject({
      kind: 'set',
      fileName: 'note.txt',
      proposedContentType: 'text/plain',
    });
    m.unmount();
  });

  test('dragover and drop on upload label apply selected binary', async () => {
    const binding = makeBinding();
    const m = mountField(binding, 'edit', { drag: true });
    await flushPromises();
    const label = m.q('label.choy-binary-upload') as HTMLLabelElement | null;
    expect(label).toBeTruthy();
    const file = new File([new Uint8Array([9])], 'drop.bin', { type: 'application/octet-stream' });
    const drop = new Event('drop', { bubbles: true, cancelable: true }) as DragEvent;
    Object.defineProperty(drop, 'dataTransfer', {
      value: { files: { 0: file, length: 1, item: (i: number) => (i === 0 ? file : null) } },
    });
    label!.dispatchEvent(drop);
    await flushPromises();
    expect(binding.fieldRef().value).toMatchObject({ kind: 'set', fileName: 'drop.bin' });
    m.unmount();
  });

  test('remove clears current binary', async () => {
    const binding = makeBinding({
      value: { attachmentObjectId: 'obj-1', fileName: 'keep.bin', sizeBytes: 1536, kind: 'set' },
    });
    const m = mountField(binding);
    await flushPromises();
    const ss = m.setupState() as any;
    await ss.removeBinary(() => binding.fieldRef(), async () => {});
    expect(binding.fieldRef().value).toBeNull();
    m.unmount();
  });

  test('accept allow-list sets pickError instead of toast', async () => {
    const binding = makeBinding({ value: null });
    const m = mountField(binding, 'edit', { accept: '.pdf,application/pdf', drag: true });
    await flushPromises();
    const ss = m.setupState() as any;
    await ss.applySelectedBinary(new File([new Uint8Array([1])], 'no.txt', { type: 'text/plain' }), () => binding.fieldRef());
    expect(binding.fieldRef().value).toBeNull();
    expect(ss.pickError).toMatch(/not allowed|不允许/i);

    await ss.applySelectedBinary(
      new File([new Uint8Array([1])], 'ok.pdf', { type: 'application/pdf' }),
      () => binding.fieldRef(),
      async () => {}
    );
    expect(binding.fieldRef().value?.fileName).toBe('ok.pdf');
    expect(ss.pickError).toBe('');
    m.unmount();
  });

  test('pickErrorRule and field-errors map sync', async () => {
    const binding = makeBinding({ value: null, prop: 'Doc' });
    const errMap = ref(new Map<string, string>());
    installFieldBaseEditStub();
    installChoyButtonStub();
    const m = mountApp(BinaryField as any, {
      props: { binding, renderMode: 'form', drag: false, accept: '.pdf' },
      provide: { 'field-errors': errMap },
    });
    await flushPromises();
    const ss = m.setupState() as any;
    await ss.applySelectedBinary(new File([new Uint8Array([1])], 'no.txt', { type: 'text/plain' }), () => binding.fieldRef());
    expect(errMap.value.get('Doc')).toMatch(/not allowed|不允许/i);
    let ruleErr: Error | undefined;
    ss.pickErrorRule.validator(null, null, (e?: Error) => {
      ruleErr = e;
    });
    expect(ruleErr).toBeTruthy();
    await ss.applySelectedBinary(
      new File([new Uint8Array([1])], 'ok.pdf', { type: 'application/pdf' }),
      () => binding.fieldRef(),
      async () => {}
    );
    expect(errMap.value.has('Doc')).toBe(false);
    ruleErr = undefined;
    ss.pickErrorRule.validator(null, null, (e?: Error) => {
      ruleErr = e;
    });
    expect(ruleErr).toBeUndefined();
    m.unmount();
  });
});
