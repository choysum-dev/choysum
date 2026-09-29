// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, h, ref } from 'vue';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import BinaryField from './ChoyBinaryField.vue';
import FieldBase from './FieldBase.vue';

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
    expect(m.q('.o-binary-current')).toBeTruthy();
    m.unmount();
  });

  test('covers objectId-only and downloadUrl-only attachment resolution', async () => {
    const objectOnly = makeBinding({
      value: { attachmentObjectId: '  obj-only  ', kind: 'set' },
    });
    const objectMount = mountField(objectOnly);
    expect(objectMount.q('.o-binary-current')).toBeTruthy();
    objectMount.unmount();
    restoreSfc(FieldBase as any);

    const objectIdAlias = makeBinding({
      value: { objectId: '  alias-obj  ', kind: 'set' },
    });
    const aliasMount = mountField(objectIdAlias);
    expect(aliasMount.q('.o-binary-current')).toBeTruthy();
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
    expect(stringMount.q('.o-binary-current')).toBeTruthy();
    stringMount.unmount();
    restoreSfc(FieldBase as any);

    const clearBinding = makeBinding({ value: { kind: 'clear' } });
    const clearMount = mountField(clearBinding);
    expect(clearMount.q('.o-binary-current')).toBeFalsy();
    clearMount.unmount();
    restoreSfc(FieldBase as any);

    const noopBinding = makeBinding({ value: { kind: 'noop' } });
    const noopMount = mountField(noopBinding);
    expect(noopMount.q('.o-binary-current')).toBeFalsy();
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
});
