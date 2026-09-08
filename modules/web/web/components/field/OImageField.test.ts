// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount wiring for OImageField. Pure limit helpers live in imageFieldLimits.test.ts.
 */

import { computed, defineComponent, h, ref } from 'vue';
import { ElMessage } from 'element-plus';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OFieldBase from './OFieldBase.vue';
import OImageField from './OImageField.vue';

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
  } as UseField;
}

const uploadOnChange: { fn: null | ((file: any) => Promise<void> | void) } = { fn: null };
const messageErrors: string[] = [];
const originalElMessageError = ElMessage.error;

function installFieldBaseEditStub() {
  stubSfc(OFieldBase as any, {
    name: 'OFieldBase',
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

function installFieldBaseDisplayStub() {
  stubSfc(OFieldBase as any, {
    name: 'OFieldBase',
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
  uploadOnChange.fn = null;
  return mountApp(OImageField as any, {
    props: {
      binding,
      renderMode: mode === 'display' ? 'display' : 'form',
      uploadProps: { drag: false, showFileList: false },
    },
    stubs: {
      'el-upload': defineComponent({
        name: 'ElUploadStub',
        props: {
          onChange: { type: Function, default: undefined },
        },
        setup(props) {
          uploadOnChange.fn = props.onChange as any;
          return () => h('div', { class: 'el-upload-stub' });
        },
      }),
      'el-button': defineComponent({
        name: 'ElButtonStub',
        setup(_, { slots }) {
          return () => h('button', { class: 'btn' }, slots.default?.());
        },
      }),
      'el-icon': defineComponent({
        name: 'ElIconStub',
        setup(_, { slots }) {
          return () => h('i', {}, slots.default?.());
        },
      }),
    },
  });
}

describe('OImageField mount wiring', () => {
  afterEach(() => {
    restoreSfc(OFieldBase as any);
    (ElMessage as any).error = originalElMessageError;
    messageErrors.length = 0;
    delete (globalThis as any).createImageBitmap;
  });

  test('rejects oversized upload via ElMessage and keeps value empty', async () => {
    (ElMessage as any).error = (msg: string) => {
      messageErrors.push(String(msg));
    };
    const binding = makeBinding({
      meta: { maxUploadBytes: 100, maxWidth: 50, maxHeight: 50 },
      store: {
        getFieldMeta: () => ({ maxUploadBytes: 100, maxWidth: 50, maxHeight: 50 }),
      },
    });
    const m = mountField(binding);
    expect(uploadOnChange.fn).toBeTruthy();
    await uploadOnChange.fn!({ raw: makeFile(200), name: 'photo.png', status: 'ready' });
    await flushPromises();
    expect(messageErrors.length).toBe(1);
    expect(messageErrors[0]).toMatch(/100 B|maximum size/i);
    expect(binding.fieldRef().value).toBeNull();
    m.unmount();
  });

  test('rejects oversized width using createImageBitmap probe', async () => {
    (ElMessage as any).error = (msg: string) => {
      messageErrors.push(String(msg));
    };
    (globalThis as any).createImageBitmap = async () => ({ width: 80, height: 20, close() {} });
    const binding = makeBinding({
      meta: { maxUploadBytes: 10_000, maxWidth: 40, maxHeight: 30 },
    });
    const m = mountField(binding);
    await uploadOnChange.fn!({ raw: makeFile(20), name: 'photo.png', status: 'ready' });
    await flushPromises();
    expect(messageErrors[0]).toMatch(/40/);
    expect(binding.fieldRef().value).toBeNull();
    m.unmount();
  });

  test('accepts valid image and writes pending attachment value', async () => {
    (ElMessage as any).error = (msg: string) => {
      messageErrors.push(String(msg));
    };
    (globalThis as any).createImageBitmap = async () => ({ width: 20, height: 20, close() {} });
    const binding = makeBinding({
      meta: { maxUploadBytes: 10_000, maxWidth: 200, maxHeight: 200 },
      store: {
        getFieldMeta: (name: string) =>
          name === 'Photo' ? { maxUploadBytes: 10_000, maxWidth: 200, maxHeight: 200 } : undefined,
      },
    });
    const m = mountField(binding);
    await uploadOnChange.fn!({ raw: makeFile(32), name: 'photo.png', status: 'ready' });
    await flushPromises();
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
    expect(m.q('.o-image-current')).toBeTruthy();
    m.unmount();
    restoreSfc(OFieldBase as any);

    const aliasMount = mountField(makeBinding({ value: { objectId: '  alias-img  ', kind: 'set' } }));
    expect(aliasMount.q('.o-image-current')).toBeTruthy();
    aliasMount.unmount();
    restoreSfc(OFieldBase as any);

    const previewMount = mountField(
      makeBinding({
        value: { previewUrl: '  /preview/img.png  ', fileName: 'img.png', kind: 'set' },
      })
    );
    await flushPromises();
    expect(previewMount.q('.o-image-current__preview')?.getAttribute('src')).toBe('/preview/img.png');
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
      restoreSfc(OFieldBase as any);
    }
  });

  test('treats string attachment values and clear/noop kinds', async () => {
    const stringMount = mountField(makeBinding({ value: '  photo.png  ' }));
    expect(stringMount.q('.o-image-current')).toBeTruthy();
    stringMount.unmount();
    restoreSfc(OFieldBase as any);

    const clearMount = mountField(makeBinding({ value: { kind: 'CLEAR' } }));
    expect(clearMount.q('.o-image-current')).toBeFalsy();
    clearMount.unmount();
    restoreSfc(OFieldBase as any);

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
});
