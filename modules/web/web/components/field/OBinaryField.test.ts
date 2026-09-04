// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { computed, defineComponent, h, ref } from 'vue';
import type { UseField } from '@/web/web/composables/useField';
import OBinaryField from './OBinaryField.vue';

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  return {
    ...actual,
    ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
  };
});

function makeBinding(opts?: { value?: unknown }): UseField {
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
  } as UseField;
}

const ElUploadStub = defineComponent({
  name: 'ElUploadStub',
  props: {
    onChange: { type: Function, default: undefined },
  },
  setup(props, { expose }) {
    expose({
      async trigger(file: File) {
        await props.onChange?.({ raw: file, name: file.name, status: 'ready' });
      },
    });
    return () => h('div', { class: 'el-upload-stub' });
  },
});

const OFieldBaseStub = defineComponent({
  name: 'OFieldBase',
  props: {
    binding: { type: Object, required: false },
  },
  setup(props, { slots }) {
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

describe('OBinaryField normalize helpers', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  async function mountField(binding: UseField) {
    return mount(OBinaryField as any, {
      props: {
        binding,
        renderMode: 'form',
        uploadProps: { drag: false, showFileList: false },
      },
      global: {
        stubs: {
          OFieldBase: OFieldBaseStub,
          'el-upload': ElUploadStub,
          'el-button': { template: '<button class="btn"><slot /></button>' },
          'el-icon': { template: '<i><slot /></i>' },
          Document: true,
          UploadFilled: true,
        },
      },
    });
  }

  it('renders attachment metadata via normalizeOptionalString helpers', async () => {
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
    const wrapper = await mountField(binding);
    await flushPromises();
    expect(wrapper.text()).toContain('report.pdf');
    expect(wrapper.find('.o-binary-current').exists()).toBe(true);
  });

  it('covers objectId-only and downloadUrl-only attachment resolution', async () => {
    const objectOnly = makeBinding({
      value: { attachmentObjectId: '  obj-only  ', kind: 'set' },
    });
    const objectWrapper = await mountField(objectOnly);
    expect(objectWrapper.find('.o-binary-current').exists()).toBe(true);

    const objectIdAlias = makeBinding({
      value: { objectId: '  alias-obj  ', kind: 'set' },
    });
    expect((await mountField(objectIdAlias)).find('.o-binary-current').exists()).toBe(true);

    const urlFallback = makeBinding({
      value: { url: '  /files/via-url.bin  ', kind: 'set' },
    });
    const urlWrapper = await mount(OBinaryField as any, {
      props: {
        binding: urlFallback,
        renderMode: 'display',
        uploadProps: { drag: false, showFileList: false },
      },
      global: {
        stubs: {
          OFieldBase: defineComponent({
            name: 'OFieldBase',
            props: { binding: { type: Object, required: false } },
            setup(props, { slots }) {
              return () =>
                h(
                  'div',
                  slots.display?.({
                    fieldValue: () => (props.binding as UseField | undefined)?.fieldRef?.() ?? ref(null),
                    renderMode: 'form',
                  })
                );
            },
          }),
          'el-upload': ElUploadStub,
          'el-button': { template: '<button class="btn"><slot /></button>' },
          'el-icon': { template: '<i><slot /></i>' },
          Document: true,
          UploadFilled: true,
        },
      },
    });
    await flushPromises();
    expect(urlWrapper.html()).toContain('/files/via-url.bin');

    const downloadOnly = makeBinding({
      value: { downloadUrl: '  /files/only.bin  ', kind: 'set' },
    });
    const downloadWrapper = await mount(OBinaryField as any, {
      props: {
        binding: downloadOnly,
        renderMode: 'display',
        uploadProps: { drag: false, showFileList: false },
      },
      global: {
        stubs: {
          OFieldBase: defineComponent({
            name: 'OFieldBase',
            props: { binding: { type: Object, required: false } },
            setup(props, { slots }) {
              return () =>
                h(
                  'div',
                  slots.display?.({
                    fieldValue: () => (props.binding as UseField | undefined)?.fieldRef?.() ?? ref(null),
                    renderMode: 'form',
                  })
                );
            },
          }),
          'el-upload': ElUploadStub,
          'el-button': { template: '<button class="btn"><slot /></button>' },
          'el-icon': { template: '<i><slot /></i>' },
          Document: true,
          UploadFilled: true,
        },
      },
    });
    await flushPromises();
    expect(downloadWrapper.html()).toContain('/files/only.bin');
  });

  it('treats string values and clear/noop kinds via hasAttachment', async () => {
    const stringBinding = makeBinding({ value: '  plain-name.bin  ' });
    const stringWrapper = await mountField(stringBinding);
    expect(stringWrapper.find('.o-binary-current').exists()).toBe(true);

    const clearBinding = makeBinding({ value: { kind: 'clear' } });
    const clearWrapper = await mountField(clearBinding);
    expect(clearWrapper.find('.o-binary-current').exists()).toBe(false);

    const noopBinding = makeBinding({ value: { kind: 'noop' } });
    const noopWrapper = await mountField(noopBinding);
    expect(noopWrapper.find('.o-binary-current').exists()).toBe(false);
  });

  it('writes pending set envelope with normalized file metadata on upload', async () => {
    const binding = makeBinding();
    const wrapper = await mountField(binding);
    const upload = wrapper.findComponent({ name: 'ElUploadStub' });
    const onChange = upload.props('onChange') as (file: any) => Promise<void>;
    const file = new File([new Uint8Array([1, 2])], 'note.txt', { type: 'text/plain' });
    await onChange({ raw: file, name: file.name, status: 'ready' });
    await flushPromises();

    expect(binding.fieldRef().value).toMatchObject({
      kind: 'set',
      fileName: 'note.txt',
      originalFileName: 'note.txt',
      proposedFileName: 'note.txt',
      proposedContentType: 'text/plain',
      clientContentType: 'text/plain',
      displayName: 'note.txt',
    });
  });
});
