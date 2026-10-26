// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount wiring for ChoyImageField. Pure limit helpers live in imageFieldLimits.test.ts.
 */

import { computed, h, ref } from 'vue';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyFieldBase from './ChoyFieldBase.vue';
import ImageField from './ChoyImageField.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { ChoyDialog, ChoyDialogContent } from '@/web/web/components/layout/choyDialog';

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

function installDialogStubs() {
  stubSfc(ChoyDialog as any, {
    name: 'ChoyDialog',
    props: { open: { type: Boolean, default: false } },
    emits: ['update:open'],
    setup(p: any, { slots }: any) {
      return () => (p.open ? h('div', { class: 'choy-dialog-stub' }, slots.default?.()) : null);
    },
  });
  stubSfc(ChoyDialogContent as any, {
    name: 'ChoyDialogContent',
    setup(_p: any, { attrs, slots }: any) {
      return () => h('div', { class: ['choy-dialog-content-stub', 'choy-image-preview-dialog'], ...attrs }, slots.default?.());
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
  installDialogStubs();
  return mountApp(ImageField as any, {
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

describe('ChoyImageField mount wiring', () => {
  afterEach(() => {
    restoreSfc(ChoyFieldBase as any);
    restoreSfc(ChoyButton as any);
    restoreSfc(ChoyDialog as any);
    restoreSfc(ChoyDialogContent as any);
    delete (globalThis as any).createImageBitmap;
  });

  test('rejects oversized upload via pickError and keeps value empty', async () => {
    const binding = makeBinding({
      meta: { maxUploadBytes: 100, maxWidth: 50, maxHeight: 50 },
      store: {
        getFieldMeta: () => ({ maxUploadBytes: 100, maxWidth: 50, maxHeight: 50 }),
      },
    });
    const m = mountField(binding);
    await pickFile(m, makeFile(200));
    const ss = m.setupState() as any;
    expect(ss.pickError).toMatch(/100 B|maximum size|上限/i);
    expect(binding.fieldRef().value).toBeNull();
    m.unmount();
  });

  test('rejects oversized width using createImageBitmap probe', async () => {
    (globalThis as any).createImageBitmap = async () => ({ width: 80, height: 20, close() {} });
    const binding = makeBinding({
      meta: { maxUploadBytes: 10_000, maxWidth: 40, maxHeight: 30 },
    });
    const m = mountField(binding);
    await pickFile(m, makeFile(20));
    expect((m.setupState() as any).pickError).toMatch(/40/);
    expect(binding.fieldRef().value).toBeNull();
    m.unmount();
  });

  test('accepts valid image and writes pending attachment value', async () => {
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
    expect((m.setupState() as any).pickError).toBe('');
    expect(binding.fieldRef().value).toMatchObject({
      kind: 'set',
      fileName: 'photo.png',
      proposedContentType: 'image/png',
    });
    m.unmount();
  });

  test('rejects non-raster image uploads such as SVG via pickError', async () => {
    const binding = makeBinding();
    const m = mountField(binding);
    const svg = new File([new Uint8Array(12)], 'icon.svg', { type: 'image/svg+xml' });
    await pickFile(m, svg);
    expect((m.setupState() as any).pickError).toMatch(/raster/i);
    expect(binding.fieldRef().value).toBeNull();
    m.unmount();
  });

  test('preview img error falls back to Picture icon', async () => {
    const binding = makeBinding({
      value: { previewUrl: '  /broken.png  ', fileName: 'img.png', kind: 'set' },
    });
    const m = mountField(binding);
    await flushPromises();
    const img = m.q('.choy-image-current__preview') as HTMLImageElement | null;
    expect(img?.getAttribute('src')).toBe('/broken.png');
    img!.dispatchEvent(new Event('error'));
    await flushPromises();
    expect(m.q('.choy-image-current__preview')).toBeFalsy();
    const ss = m.setupState() as any;
    expect(ss.previewSrc(binding.fieldRef().value)).toBeUndefined();
    const svg = Array.from(m.el.querySelectorAll('svg')).find(el => el.getAttribute('data-lucide') === 'Image');
    expect(svg).toBeTruthy();
    m.unmount();
  });

  test('preview img error on internal binding url retries with authenticated data url', async () => {
    const { setTokenProvider } = await import('@/core/web/rpc');
    setTokenProvider({
      getToken: async () => 'img-token',
      refreshToken: async () => true,
      shouldRefreshToken: async () => false,
    });
    const png = new Uint8Array([0x89, 0x50, 0x4e, 0x47]);
    (globalThis as any).fetch = async () => ({
      ok: true,
      headers: { get: () => 'image/png' },
      blob: async () => new Blob([png], { type: 'image/png' }),
    });

    const binding = makeBinding({
      value: {
        previewUrl: '/_document/bindings/bind-1/content?token=stale',
        fileName: 'img.png',
        kind: 'attachment',
      },
    });
    const m = mountField(binding);
    await flushPromises();
    const img = m.q('.choy-image-current__preview') as HTMLImageElement | null;
    expect(img?.getAttribute('src')).toContain('/_document/bindings/bind-1/content');
    img!.dispatchEvent(new Event('error'));
    await flushPromises();
    const ss = m.setupState() as any;
    const src = ss.previewSrc(binding.fieldRef().value);
    expect(src).toMatch(/^data:image\/png;base64,/);
    expect(m.q('.choy-image-current__preview')?.getAttribute('src')).toMatch(/^data:image\/png;base64,/);
    m.unmount();
    delete (globalThis as any).fetch;
    setTokenProvider({
      getToken: async () => null,
      refreshToken: async () => false,
      shouldRefreshToken: async () => false,
    });
  });

  test('resolves object id and preview urls for existing attachments', async () => {
    const previewMount = mountField(
      makeBinding({
        value: { previewUrl: '  /preview/img.png  ', fileName: 'img.png', kind: 'set' },
      })
    );
    await flushPromises();
    expect(previewMount.q('.choy-image-current__preview')?.getAttribute('src')).toBe('/preview/img.png');
    expect(previewMount.q('.choy-image-current__preview')?.getAttribute('alt')).toBe('img.png');
    previewMount.unmount();
  });

  test('form edit uses image avatar chrome without filename meta chip', async () => {
    const binding = makeBinding({
      value: {
        previewUrl: '/preview/img.png',
        fileName: 'img.png',
        mimeType: 'image/png',
        sizeBytes: 2048,
        kind: 'set',
        downloadUrl: '/preview/img.png',
      },
    });
    const m = mountField(binding);
    await flushPromises();
    const avatar = m.q('.choy-image-avatar') as HTMLElement | null;
    expect(avatar).toBeTruthy();
    expect(avatar?.getAttribute('data-fit')).toBe('contain');
    expect(avatar?.className).toMatch(/rounded-md/);
    expect(avatar?.className).not.toMatch(/rounded-xl/);
    expect(m.q('.choy-attachment-current')).toBeFalsy();
    expect(m.text()).not.toMatch(/image\/png/);
    const overlay = m.q('.choy-image-avatar__overlay') as HTMLElement | null;
    expect(overlay).toBeTruthy();
    const lucide = (name: string) =>
      Array.from(overlay?.querySelectorAll('svg') || []).some(el => el.getAttribute('data-lucide') === name);
    expect(lucide('Download')).toBe(true);
    expect(lucide('RefreshCw')).toBe(true);
    expect(lucide('Trash2')).toBe(true);
    expect(overlay?.querySelectorAll('[data-slot="tooltip"]').length).toBeGreaterThanOrEqual(2);
    m.unmount();
  });

  test('form contain fit sizes avatar from thumbnail natural ratio', async () => {
    const m = mountField(
      makeBinding({
        value: { previewUrl: '/preview/wide.png', fileName: 'wide.png', kind: 'set' },
      }),
      'display'
    );
    await flushPromises();
    const img = m.q('.choy-image-current__preview') as HTMLImageElement | null;
    expect(img).toBeTruthy();
    Object.defineProperty(img, 'naturalWidth', { configurable: true, value: 1600 });
    Object.defineProperty(img, 'naturalHeight', { configurable: true, value: 900 });
    img!.dispatchEvent(new Event('load'));
    await flushPromises();
    const avatar = m.q('button.choy-image-avatar') as HTMLElement | null;
    expect(avatar?.style.width).toBe('112px');
    expect(avatar?.style.height).toBe('63px');
    expect(avatar?.getAttribute('data-fit')).toBe('contain');
    m.unmount();
  });

  test('fit=cover keeps square crop box; radius prop applies', async () => {
    const m = mountField(
      makeBinding({
        value: { previewUrl: '/preview/img.png', fileName: 'photo.png', kind: 'set' },
      }),
      'display',
      { fit: 'cover', radius: 'none', width: 96 }
    );
    await flushPromises();
    const avatar = m.q('button.choy-image-avatar') as HTMLElement | null;
    expect(avatar?.getAttribute('data-fit')).toBe('cover');
    expect(avatar?.className).toMatch(/rounded-none/);
    expect(avatar?.style.width).toBe('96px');
    expect(avatar?.style.height).toBe('96px');
    m.unmount();
  });

  test('form display opens in-page preview dialog instead of download', async () => {
    const m = mountField(
      makeBinding({
        value: {
          previewUrl: '/preview/img.png',
          fileName: 'photo.png',
          mimeType: 'image/png',
          kind: 'set',
          downloadUrl: '/files/photo.png',
        },
      }),
      'display'
    );
    await flushPromises();
    const avatar = m.q('button.choy-image-avatar') as HTMLButtonElement | null;
    expect(avatar).toBeTruthy();
    expect(avatar?.getAttribute('href')).toBeFalsy();
    expect(m.q('.choy-attachment-display-card')).toBeFalsy();
    expect(m.q('.choy-dialog-stub')).toBeFalsy();
    avatar!.dispatchEvent(new Event('click', { bubbles: true, cancelable: true }));
    await flushPromises();
    const ss = m.setupState() as any;
    expect(ss.previewDialogOpen).toBe(true);
    expect(ss.previewDialogSrc).toBe('/preview/img.png');
    expect(m.q('.choy-dialog-stub')).toBeTruthy();
    const img = m.q('.choy-image-preview-dialog__img') as HTMLImageElement | null;
    expect(img?.getAttribute('src')).toBe('/preview/img.png');
    expect(m.q('.choy-image-preview-dialog__shell')).toBeTruthy();
    Object.defineProperty(img, 'naturalWidth', { configurable: true, value: 1600 });
    Object.defineProperty(img, 'naturalHeight', { configurable: true, value: 900 });
    img!.dispatchEvent(new Event('load'));
    await flushPromises();
    expect(ss.previewDialogNatural).toEqual({ width: 1600, height: 900 });
    expect(img!.style.width).toMatch(/px$/);
    expect(img!.style.height).toMatch(/px$/);
    const shell = m.q('.choy-image-preview-dialog__shell') as HTMLElement | null;
    expect(shell?.style.width).toMatch(/px$/);
    expect(shell?.style.height).toMatch(/px$/);
    m.unmount();
  });

  test('form avatar honors width/height props', async () => {
    const m = mountField(
      makeBinding({
        value: { previewUrl: '/preview/img.png', fileName: 'photo.png', kind: 'set' },
      }),
      'display',
      { width: 160, height: 90 }
    );
    await flushPromises();
    const avatar = m.q('button.choy-image-avatar') as HTMLElement | null;
    expect(avatar).toBeTruthy();
    expect(avatar?.style.width).toBe('160px');
    expect(avatar?.style.height).toBe('90px');
    m.unmount();
  });

  test('empty display uses choy-field-display-empty', async () => {
    const m = mountField(makeBinding({ value: null }), 'display');
    await flushPromises();
    expect(m.q('.choy-field-display-empty')).toBeTruthy();
    m.unmount();
  });

  test('dragover and drop on upload label apply selected image', async () => {
    (globalThis as any).createImageBitmap = async () => ({ width: 20, height: 20, close() {} });
    const binding = makeBinding({
      meta: { maxUploadBytes: 10_000, maxWidth: 200, maxHeight: 200 },
    });
    const m = mountField(binding, 'edit', { drag: true });
    await flushPromises();
    const label = m.q('label.choy-image-upload') as HTMLLabelElement | null;
    expect(label).toBeTruthy();
    const file = makeFile(32);
    const drop = new Event('drop', { bubbles: true, cancelable: true }) as DragEvent;
    Object.defineProperty(drop, 'dataTransfer', {
      value: { files: { 0: file, length: 1, item: (i: number) => (i === 0 ? file : null) } },
    });
    label!.dispatchEvent(drop);
    await flushPromises();
    expect(binding.fieldRef().value).toMatchObject({ kind: 'set', fileName: 'photo.png' });
    m.unmount();
  });

  test('remove clears current image', async () => {
    const binding = makeBinding({
      value: { attachmentObjectId: 'obj-1', fileName: 'keep.png', kind: 'set' },
    });
    const m = mountField(binding);
    await flushPromises();
    const ss = m.setupState() as any;
    await ss.removeImage(() => binding.fieldRef(), async () => {});
    expect(binding.fieldRef().value).toBeNull();
    m.unmount();
  });

  test('createLocalPreview returns data URL or undefined on FileReader failure', async () => {
    const binding = makeBinding({
      value: { fileName: 'x.png', kind: 'set' },
    });
    const m = mountField(binding);
    await flushPromises();
    const ss = m.setupState() as any;
    const OriginalFR = (globalThis as any).FileReader;
    (globalThis as any).FileReader = class {
      result: string | ArrayBuffer | null = null;
      onload: (() => void) | null = null;
      onerror: (() => void) | null = null;
      onabort: (() => void) | null = null;
      readAsDataURL(_file: Blob) {
        this.result = 'data:image/png;base64,xxx';
        this.onload?.();
      }
    };
    expect(await ss.createLocalPreview(makeFile(8))).toBe('data:image/png;base64,xxx');
    (globalThis as any).FileReader = OriginalFR;
    m.unmount();
  });
});
