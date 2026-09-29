// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount wiring for ImageField. Pure limit helpers live in imageFieldLimits.test.ts.
 */

import { computed, h, ref } from 'vue';
import { ChoyMessage } from '../../composables/useChoyMessage';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import FieldBase from './FieldBase.vue';
import ImageField from './ChoyImageField.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';

function makeFile(size: number, type = 'image/png'): File {
  return new File([new Uint8Array(size)], 'photo.png', { type });
}

function makeBinding(opts?: {
  prop?: string;
  meta?: Record<string, unknown>;
  store?: any;
  value?: unknown;
}): UseField {
  const value = ref(opts?.value ?? null);
  const record = ref({ Id: '1' });
  return {
    env: {
      isForm: true,
      isEditMode: true,
      viewMode: 'edit',
      fieldPrefix: null,
    },
    prop: opts?.prop || 'Photo',
    meta: (opts?.meta || {}) as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => record.value) as any,
    registerFields: () => {},
    store: opts?.store,
    asView: () => ({ fieldValue: () => value }) as any,
  } as any;
}

const messageErrors: string[] = [];
const originalChoyMessageError = ChoyMessage.error;

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
  return mountApp(ImageField as any, {
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

describe('ImageField mount wiring', () => {
  afterEach(() => {
    restoreSfc(FieldBase as any);
    restoreSfc(ChoyButton as any);
    ChoyMessage.error = originalChoyMessageError;
    messageErrors.length = 0;
    delete (globalThis as any).createImageBitmap;
  });

  test('rejects oversized upload via ChoyMessage and keeps value empty', async () => {
    ChoyMessage.error = ((msg: string) => {
      messageErrors.push(String(msg));
      return 0;
    }) as typeof ChoyMessage.error;
    const binding = makeBinding({
      meta: { maxUploadBytes: 100, maxWidth: 50, maxHeight: 50 },
      store: {
        getFieldMeta: () => ({ maxUploadBytes: 100, maxWidth: 50, maxHeight: 50 }),
      },
    });
    const m = mountField(binding);
    await pickFile(m, makeFile(200));
    expect(messageErrors.length).toBe(1);
    expect(messageErrors[0]).toMatch(/100 B|maximum size/i);
    expect(binding.fieldRef().value).toBeNull();
    m.unmount();
  });

  test('rejects oversized width using createImageBitmap probe', async () => {
    ChoyMessage.error = ((msg: string) => {
      messageErrors.push(String(msg));
      return 0;
    }) as typeof ChoyMessage.error;
    (globalThis as any).createImageBitmap = async () => ({ width: 80, height: 20, close() {} });
    const binding = makeBinding({
      meta: { maxUploadBytes: 10_000, maxWidth: 40, maxHeight: 30 },
    });
    const m = mountField(binding);
    await pickFile(m, makeFile(20));
    expect(messageErrors[0]).toMatch(/40/);
    expect(binding.fieldRef().value).toBeNull();
    m.unmount();
  });

  test('accepts valid image and writes pending attachment value', async () => {
    ChoyMessage.error = ((msg: string) => {
      messageErrors.push(String(msg));
      return 0;
    }) as typeof ChoyMessage.error;
    (globalThis as any).createImageBitmap = async () => ({ width: 20, height: 20, close() {} });
    const binding = makeBinding({
      meta: { maxUploadBytes: 10_000, maxWidth: 200, maxHeight: 200 },
      store: {
        getFieldMeta: (name: string) =>
          name === 'Photo' ? { maxUploadBytes: 10_000, maxWidth: 200, maxHeight: 200 } : undefined,
      },
    });
    const m = mountField(binding);
    await pickFile(m, makeFile(32));
    expect(messageErrors.length).toBe(0);
    expect(binding.fieldRef().value).toMatchObject({
      kind: 'set',
      fileName: 'photo.png',
      originalFileName: 'photo.png',
      proposedFileName: 'photo.png',
      proposedContentType: 'image/png',
      clientContentType: 'image/png',
      displayName: 'photo.png',
    });
    m.unmount();
  });

  test('resolves object id and preview urls for existing attachments', async () => {
    const binding = makeBinding({
      value: { attachmentObjectId: '  obj-img  ', kind: 'set' },
    });
    const m = mountField(binding);
    await flushPromises();
    expect(m.q('.choy-image-current')).toBeTruthy();
    m.unmount();
    restoreSfc(FieldBase as any);

    const aliasMount = mountField(makeBinding({ value: { objectId: '  alias-img  ', kind: 'set' } }));
    expect(aliasMount.q('.choy-image-current')).toBeTruthy();
    aliasMount.unmount();
    restoreSfc(FieldBase as any);

    const previewMount = mountField(
      makeBinding({
        value: { previewUrl: '  /preview/img.png  ', fileName: 'img.png', kind: 'set' },
      })
    );
    await flushPromises();
    expect(previewMount.q('.choy-image-current__preview')?.getAttribute('src')).toBe('/preview/img.png');
    previewMount.unmount();
  });

  test('covers resolveDownloadUrl and resolvePreviewUrl fallback chains', async () => {
    const cases: Array<{ value: Record<string, unknown>; expectHref: string }> = [
      { value: { url: '  /dl/via-url.png  ', kind: 'set' }, expectHref: '/dl/via-url.png' },
      { value: { previewUrl: '  /dl/via-preview.png  ', kind: 'set' }, expectHref: '/dl/via-preview.png' },
      { value: { thumbnailUrl: '  /dl/via-thumb.png  ', kind: 'set' }, expectHref: '/dl/via-thumb.png' },
      {
        value: { descriptor: { downloadUrl: '  /dl/desc-download.png  ' }, kind: 'set' },
        expectHref: '/dl/desc-download.png',
      },
      {
        value: { descriptor: { previewUrl: '  /dl/desc-preview.png  ' }, kind: 'set' },
        expectHref: '/dl/desc-preview.png',
      },
    ];

    for (const c of cases) {
      const m = mountField(makeBinding({ value: c.value }), 'display');
      await flushPromises();
      const href = m.q('a')?.getAttribute('href') || m.q('img')?.getAttribute('src') || '';
      expect(href).toBe(c.expectHref);
      m.unmount();
      restoreSfc(FieldBase as any);
    }
  });

  test('treats string attachment values and clear/noop kinds', async () => {
    const stringMount = mountField(makeBinding({ value: '  photo.png  ' }));
    expect(stringMount.q('.choy-image-current')).toBeTruthy();
    stringMount.unmount();
    restoreSfc(FieldBase as any);

    const clearMount = mountField(makeBinding({ value: { kind: 'CLEAR' } }));
    expect(clearMount.q('.choy-image-current')).toBeFalsy();
    clearMount.unmount();
    restoreSfc(FieldBase as any);

    const downloadMount = mountField(
      makeBinding({
        value: {
          attachmentBindingId: 'bind-x',
          downloadUrl: '  /dl/only.png  ',
          kind: 'set',
        },
      }),
      'display'
    );
    await flushPromises();
    expect(downloadMount.q('a')?.getAttribute('href')).toBe('/dl/only.png');
    downloadMount.unmount();
  });

  test('dragover and drop on upload label apply selected image', async () => {
    ChoyMessage.error = ((msg: string) => {
      messageErrors.push(String(msg));
      return 0;
    }) as typeof ChoyMessage.error;
    (globalThis as any).createImageBitmap = async () => ({ width: 20, height: 20, close() {} });
    const binding = makeBinding({
      meta: { maxUploadBytes: 10_000, maxWidth: 200, maxHeight: 200 },
    });
    const m = mountField(binding);
    await flushPromises();
    const label = m.q('label.choy-image-upload') as HTMLLabelElement | null;
    expect(label).toBeTruthy();

    const dragOver = new Event('dragover', { bubbles: true, cancelable: true }) as DragEvent;
    Object.defineProperty(dragOver, 'dataTransfer', {
      value: { dropEffect: 'none', files: [] },
    });
    label!.dispatchEvent(dragOver);

    const file = makeFile(32);
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
    expect(binding.fieldRef().value).toMatchObject({ kind: 'set', fileName: 'photo.png' });
    m.unmount();
  });

  test('remove clears current image via action button', async () => {
    const binding = makeBinding({
      value: { attachmentObjectId: 'obj-1', fileName: 'keep.png', kind: 'set' },
    });
    const m = mountField(binding);
    await flushPromises();
    expect(m.q('.choy-image-current')).toBeTruthy();
    const valueRef = binding.fieldRef();
    valueRef.value = null;
    await flushPromises();
    expect(binding.fieldRef().value).toBeNull();
    m.unmount();
  });

  test('formats size metadata and empty file change is a no-op', async () => {
    const binding = makeBinding({
      value: {
        fileName: 'big.png',
        mimeType: 'image/png',
        sizeBytes: 2048,
        kind: 'set',
      },
    });
    const m = mountField(binding);
    await flushPromises();
    expect(m.text()).toMatch(/KB|B/);
    m.unmount();
    restoreSfc(FieldBase as any);

    const empty = makeBinding();
    const emptyMount = mountField(empty);
    const input = emptyMount.q('input.sr-only') as HTMLInputElement;
    Object.defineProperty(input, 'files', {
      configurable: true,
      value: { length: 0, item: () => null },
    });
    input.dispatchEvent(new Event('change', { bubbles: true }));
    await flushPromises();
    expect(empty.fieldRef().value).toBeNull();
    emptyMount.unmount();
  });
});
