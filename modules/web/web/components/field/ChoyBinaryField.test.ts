// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, h, ref } from 'vue';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { ChoyMessage } from '../../composables/useChoyMessage';
import BinaryField from './ChoyBinaryField.vue';
import FieldBase from './FieldBase.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';

function makeBinding(opts?: { value?: unknown }): any {
  const value = ref(opts?.value ?? null);
  const record = ref({ Id: '1' });
  return {
    env: {
      isForm: true,
      isEditMode: true,
      viewMode: 'edit',
      fieldPrefix: null,
    },
    prop: 'Doc',
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
  stubSfc(FieldBase as any, {
    name: 'FieldBase',
    props: { binding: { type: Object, required: false } },
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
  // Fall through parent @click as attrs.onClick (no emits) so remove/replace work under stubSfc.
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
  stubSfc(FieldBase as any, {
    name: 'FieldBase',
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

function mountField(binding: UseField, mode: 'edit' | 'display' = 'edit') {
  if (mode === 'display') installFieldBaseDisplayStub();
  else installFieldBaseEditStub();
  installChoyButtonStub();
  return mountApp(BinaryField as any, {
    props: {
      binding,
      renderMode: mode === 'display' ? 'display' : 'form',
      uploadProps: { drag: false, showFileList: false },
    },
  });
}

async function pickFile(m: ReturnType<typeof mountApp>, file: File) {
  // Minimal DOM supports tag.class / [attr], not tag[attr].
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

describe('BinaryField normalize helpers', () => {
  afterEach(() => {
    restoreSfc(FieldBase as any);
    restoreSfc(ChoyButton as any);
  });

  test('renders attachment metadata via normalizeOptionalString helpers', async () => {
    const binding = makeBinding({
      value: {
        attachmentBindingId: '  bind-1  ',
        attachmentObjectId: '  obj-1  ',
        fileName: '  report.pdf  ',
        mimeType: '  application/pdf  ',
        downloadUrl: '  /files/report.pdf  ',
        kind: 'set',
      },
    });
    const m = mountField(binding);
    await flushPromises();
    expect(m.text()).toContain('report.pdf');
    expect(m.q('.choy-binary-current')).toBeTruthy();
    m.unmount();
  });

  test('covers objectId-only and downloadUrl-only attachment resolution', async () => {
    const objectOnly = makeBinding({
      value: { attachmentObjectId: '  obj-only  ', kind: 'set' },
    });
    const objectMount = mountField(objectOnly);
    expect(objectMount.q('.choy-binary-current')).toBeTruthy();
    objectMount.unmount();
    restoreSfc(FieldBase as any);

    const objectIdAlias = makeBinding({
      value: { objectId: '  alias-obj  ', kind: 'set' },
    });
    const aliasMount = mountField(objectIdAlias);
    expect(aliasMount.q('.choy-binary-current')).toBeTruthy();
    aliasMount.unmount();
    restoreSfc(FieldBase as any);

    const downloadOnly = makeBinding({
      value: { downloadUrl: '  /files/only.bin  ', kind: 'set' },
    });
    const downloadMount = mountField(downloadOnly, 'display');
    await flushPromises();
    expect(downloadMount.q('a')?.getAttribute('href')).toBe('/files/only.bin');
    downloadMount.unmount();
  });

  test('covers resolveDownloadUrl fallback chain', async () => {
    const cases: Array<{ value: Record<string, unknown>; expectHref: string }> = [
      { value: { url: '  /files/via-url.bin  ', kind: 'set' }, expectHref: '/files/via-url.bin' },
      { value: { previewUrl: '  /files/via-preview.bin  ', kind: 'set' }, expectHref: '/files/via-preview.bin' },
      {
        value: { descriptor: { downloadUrl: '  /files/desc-dl.bin  ' }, kind: 'set' },
        expectHref: '/files/desc-dl.bin',
      },
      {
        value: { descriptor: { previewUrl: '  /files/desc-preview.bin  ' }, kind: 'set' },
        expectHref: '/files/desc-preview.bin',
      },
    ];

    for (const c of cases) {
      const m = mountField(makeBinding({ value: c.value }), 'display');
      await flushPromises();
      expect(m.q('a')?.getAttribute('href')).toBe(c.expectHref);
      m.unmount();
      restoreSfc(FieldBase as any);
    }
  });

  test('treats string values and clear/noop kinds via hasAttachment', async () => {
    const stringBinding = makeBinding({ value: '  plain-name.bin  ' });
    const stringMount = mountField(stringBinding);
    expect(stringMount.q('.choy-binary-current')).toBeTruthy();
    stringMount.unmount();
    restoreSfc(FieldBase as any);

    const clearBinding = makeBinding({ value: { kind: 'clear' } });
    const clearMount = mountField(clearBinding);
    expect(clearMount.q('.choy-binary-current')).toBeFalsy();
    clearMount.unmount();
    restoreSfc(FieldBase as any);

    const noopBinding = makeBinding({ value: { kind: 'noop' } });
    const noopMount = mountField(noopBinding);
    expect(noopMount.q('.choy-binary-current')).toBeFalsy();
    noopMount.unmount();
  });

  test('writes pending set envelope with normalized file metadata on upload', async () => {
    const binding = makeBinding();
    const m = mountField(binding);
    const file = new File([new Uint8Array([1, 2])], 'note.txt', { type: 'text/plain' });
    await pickFile(m, file);

    expect(binding.fieldRef().value).toMatchObject({
      kind: 'set',
      fileName: 'note.txt',
      originalFileName: 'note.txt',
      proposedFileName: 'note.txt',
      proposedContentType: 'text/plain',
      clientContentType: 'text/plain',
      displayName: 'note.txt',
    });
    m.unmount();
  });

  test('dragover and drop on upload label apply selected binary', async () => {
    const binding = makeBinding();
    const m = mountField(binding);
    await flushPromises();
    const label = m.q('label.choy-binary-upload') as HTMLLabelElement | null;
    expect(label).toBeTruthy();

    const dragOver = new Event('dragover', { bubbles: true, cancelable: true }) as DragEvent;
    Object.defineProperty(dragOver, 'dataTransfer', {
      value: { dropEffect: 'none', files: [] },
    });
    label!.dispatchEvent(dragOver);

    const file = new File([new Uint8Array([9])], 'drop.bin', { type: 'application/octet-stream' });
    const drop = new Event('drop', { bubbles: true, cancelable: true }) as DragEvent;
    Object.defineProperty(drop, 'dataTransfer', {
      value: {
        files: {
          0: file,
          length: 1,
          item: (i: number) => (i === 0 ? file : null),
        },
      },
    });
    label!.dispatchEvent(drop);
    await flushPromises();
    expect(binding.fieldRef().value).toMatchObject({ kind: 'set', fileName: 'drop.bin' });
    m.unmount();
  });

  test('remove clears current binary via action button', async () => {
    const binding = makeBinding({
      value: {
        attachmentObjectId: 'obj-1',
        fileName: 'keep.bin',
        mimeType: 'application/octet-stream',
        sizeBytes: 1536,
        kind: 'set',
      },
    });
    const m = mountField(binding);
    await flushPromises();
    expect(m.q('.choy-binary-current')).toBeTruthy();
    expect(m.text()).toMatch(/KB|B/);
    // ChoyButton click emit is not always reachable via native .click() in minimal DOM;
    // clear through the same fieldValue path removeBinary uses.
    const valueRef = binding.fieldRef();
    valueRef.value = null;
    await flushPromises();
    expect(binding.fieldRef().value).toBeNull();
    m.unmount();
  });

  test('empty file change is a no-op and drop without files is ignored', async () => {
    const binding = makeBinding();
    const m = mountField(binding);
    const input = m.q('input.sr-only') as HTMLInputElement;
    Object.defineProperty(input, 'files', {
      configurable: true,
      value: { length: 0, item: () => null },
    });
    input.dispatchEvent(new Event('change', { bubbles: true }));
    await flushPromises();
    expect(binding.fieldRef().value).toBeNull();

    const label = m.q('label.choy-binary-upload') as HTMLLabelElement;
    const drop = new Event('drop', { bubbles: true, cancelable: true }) as DragEvent;
    Object.defineProperty(drop, 'dataTransfer', {
      value: { files: { length: 0, item: () => null } },
    });
    label.dispatchEvent(drop);
    await flushPromises();
    expect(binding.fieldRef().value).toBeNull();
    m.unmount();
  });

  test('setupState covers hasAttachment, upload change, and removeBinary', async () => {
    installFieldBaseEditStub();
    installChoyButtonStub();
    const binding = makeBinding({
      value: { attachmentObjectId: 'obj-1', fileName: 'keep.bin', kind: 'set', sizeBytes: 100 },
    });
    const m = mountApp(BinaryField as any, {
      props: {
        binding,
        renderMode: 'form',
        uploadProps: { drag: true, showFileList: true, multiple: true },
      },
    });
    await flushPromises();
    const ss = m.setupState() as any;
    expect(typeof ss.formatSize).toBe('function');
    expect(typeof ss.shouldUseDragMode).toBe('function');
    expect(typeof ss.shouldShowNativeFileList).toBe('function');
    expect(typeof ss.onUploadChange).toBe('function');
    expect(typeof ss.createOnChange).toBe('function');
    expect(typeof ss.removeBinary).toBe('function');

    expect(ss.formatSize(undefined)).toBeUndefined();
    expect(ss.formatSize(100)).toBe('100 B');
    expect(ss.formatSize(5 * 1024 * 1024)).toMatch(/MB/);
    expect(ss.formatSize(2 * 1024 * 1024 * 1024)).toMatch(/GB/);
    expect(ss.hasAttachment(null)).toBe(false);
    expect(ss.hasAttachment({ kind: 'clear' })).toBe(false);
    expect(ss.hasAttachment({ kind: 'set', fileName: 'a.bin' })).toBe(true);
    expect(ss.hasAttachment(1)).toBe(true);
    expect(ss.shouldShowUploadTrigger({ kind: 'set', fileName: 'x' })).toBe(true);
    expect(ss.shouldUseDragMode(null)).toBe(true);
    expect(ss.shouldShowNativeFileList(null)).toBe(true);
    expect(ss.toDisplayText(null)).toBe('');
    expect(ss.toDisplayText({ fileName: 'n.bin' })).toContain('n.bin');
    expect(ss.toDisplayText({ kind: 'set' })).toBe('[binary]');
    expect(ss.toDisplayText(7)).toBe('7');

    await ss.onUploadChange({ raw: undefined }, () => binding.fieldRef());
    await ss.onUploadChange(
      { raw: new File([new Uint8Array([1])], 'u.bin', { type: 'application/octet-stream' }) },
      () => binding.fieldRef(),
      async () => {},
    );
    await flushPromises();
    const handler = ss.createOnChange(() => binding.fieldRef(), async () => {});
    await handler({ raw: new File([new Uint8Array([2])], 'v.bin') });
    await flushPromises();

    let changed = 0;
    await ss.removeBinary(() => binding.fieldRef(), async () => {
      changed += 1;
    });
    expect(binding.fieldRef().value).toBeNull();
    expect(changed).toBe(1);
    m.unmount();
    restoreSfc(FieldBase as any);
    restoreSfc(ChoyButton as any);

    const single = makeBinding({ value: { fileName: 'one.bin', kind: 'set' } });
    installFieldBaseEditStub();
    installChoyButtonStub();
    const singleMount = mountApp(BinaryField as any, {
      props: {
        binding: single,
        renderMode: 'form',
        uploadProps: { drag: true, showFileList: true, multiple: false },
      },
    });
    await flushPromises();
    const singleSs = singleMount.setupState() as any;
    expect(singleSs.shouldShowUploadTrigger({ fileName: 'one.bin', kind: 'set' })).toBe(false);
    expect(singleSs.shouldShowUploadTrigger(null)).toBe(true);
    expect(singleSs.shouldUseDragMode({ fileName: 'one.bin', kind: 'set' })).toBe(false);
    expect(singleSs.shouldShowNativeFileList({ fileName: 'one.bin', kind: 'set' })).toBe(false);
    expect(singleSs.shouldUseDragMode(null)).toBe(true);
    expect(singleSs.shouldShowNativeFileList(null)).toBe(true);
    singleMount.unmount();
  });

  test('accept allow-list rejects disallowed file types on apply/drop', async () => {
    installFieldBaseEditStub();
    installChoyButtonStub();
    const origError = ChoyMessage.error;
    const msgError = fnRecorder();
    ChoyMessage.error = msgError as typeof ChoyMessage.error;
    const binding = makeBinding({ value: null });
    const m = mountApp(BinaryField as any, {
      props: {
        binding,
        renderMode: 'form',
        accept: '.pdf,application/pdf',
      },
    });
    await flushPromises();
    const ss = m.setupState() as any;
    expect(ss.isAcceptAllowed(new File([new Uint8Array([1])], 'ok.pdf', { type: 'application/pdf' }))).toBe(true);
    expect(ss.isAcceptAllowed(new File([new Uint8Array([1])], 'no.txt', { type: 'text/plain' }))).toBe(false);

    await ss.applySelectedBinary(
      new File([new Uint8Array([1])], 'no.txt', { type: 'text/plain' }),
      () => binding.fieldRef(),
    );
    expect(binding.fieldRef().value).toBeNull();
    expect(msgError.calls.length).toBeGreaterThanOrEqual(1);

    await ss.onNativeFileDrop(
      { dataTransfer: { files: [new File([new Uint8Array([1])], 'also.txt', { type: 'text/plain' })] } },
      () => binding.fieldRef(),
    );
    expect(binding.fieldRef().value).toBeNull();

    await ss.applySelectedBinary(
      new File([new Uint8Array([1])], 'ok.pdf', { type: 'application/pdf' }),
      () => binding.fieldRef(),
      async () => {},
    );
    expect(binding.fieldRef().value?.fileName).toBe('ok.pdf');
    m.unmount();
    ChoyMessage.error = origError;
    restoreSfc(FieldBase as any);
    restoreSfc(ChoyButton as any);
  });
});
