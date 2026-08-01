// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { computed, defineComponent, h, ref } from 'vue';
import { DEFAULT_GLOBAL_MAX_UPLOAD_BYTES } from '@/core/service/orm/upload_limits';
import type { UseField } from '@/web/web/composables/useField';
import {
  formatImageByteLimit,
  imageFieldLimitErrorMessage,
  readImageNaturalDimensions,
  reportImageFieldValidation,
  resolveImageFieldLimits,
  resolveImageFieldLimitsFromSources,
  validateImageFieldFile,
} from './imageFieldLimits';
import OImageField from './OImageField.vue';

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  return {
    ...actual,
    ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
  };
});

function makeFile(size: number, type = 'image/png'): File {
  const buffer = new Uint8Array(size);
  return new File([buffer], 'photo.png', { type });
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

describe('imageFieldLimits (PR-P2-F3)', () => {
  const limits = {
    maxUploadBytes: 1024,
    maxWidth: 100,
    maxHeight: 80,
  };

  it('resolveImageFieldLimits caps bytes and ignores invalid values', () => {
    expect(resolveImageFieldLimits(null)).toEqual({});
    expect(resolveImageFieldLimits(undefined)).toEqual({});
    expect(resolveImageFieldLimits({ maxUploadBytes: 0, maxWidth: -1, maxHeight: 0 })).toEqual({});
    expect(resolveImageFieldLimits({ maxUploadBytes: 100, maxWidth: 10, maxHeight: 20 })).toEqual({
      maxUploadBytes: 100,
      maxWidth: 10,
      maxHeight: 20,
    });
    expect(resolveImageFieldLimits({ maxUploadBytes: DEFAULT_GLOBAL_MAX_UPLOAD_BYTES + 1 })).toEqual({
      maxUploadBytes: DEFAULT_GLOBAL_MAX_UPLOAD_BYTES,
    });
  });

  it('formatImageByteLimit covers B/KB/MB', () => {
    expect(formatImageByteLimit(500)).toBe('500 B');
    expect(formatImageByteLimit(2048)).toBe('2 KB');
    expect(formatImageByteLimit(3 * 1024 * 1024)).toBe('3 MB');
  });

  it('rejects oversized file', async () => {
    const result = await validateImageFieldFile(makeFile(2048), limits, async () => ({ width: 50, height: 50 }));
    expect(result).toEqual({ ok: false, reason: 'fileTooLarge', detail: '1 KB' });
  });

  it('rejects oversized width', async () => {
    const result = await validateImageFieldFile(makeFile(512), limits, async () => ({ width: 200, height: 50 }));
    expect(result).toEqual({ ok: false, reason: 'widthTooLarge', detail: '100' });
  });

  it('rejects oversized height', async () => {
    const result = await validateImageFieldFile(makeFile(512), limits, async () => ({ width: 50, height: 120 }));
    expect(result).toEqual({ ok: false, reason: 'heightTooLarge', detail: '80' });
  });

  it('accepts file within limits', async () => {
    const readDimensions = vi.fn(async () => ({ width: 80, height: 60 }));
    const result = await validateImageFieldFile(makeFile(512), limits, readDimensions);
    expect(result).toEqual({ ok: true });
    expect(readDimensions).toHaveBeenCalled();
  });

  it('skips dimension checks when no dimension limits or probe missing', async () => {
    expect(await validateImageFieldFile(makeFile(10), { maxUploadBytes: 100 }, async () => ({ width: 999, height: 999 }))).toEqual({
      ok: true,
    });
    expect(await validateImageFieldFile(makeFile(10), { maxWidth: 10 }, async () => undefined)).toEqual({ ok: true });
  });

  it('resolveImageFieldLimitsFromSources prefers store meta then binding meta', () => {
    expect(
      resolveImageFieldLimitsFromSources({
        bindingProp: '',
        propsProp: 'Photo',
        propsStore: {
          getFieldMeta: name => (name === 'Photo' ? { maxUploadBytes: 128, maxWidth: 9 } : undefined),
        },
        bindingMeta: { maxHeight: 3 },
      })
    ).toEqual({ maxUploadBytes: 128, maxWidth: 9 });

    expect(
      resolveImageFieldLimitsFromSources({
        bindingProp: 'Photo',
        bindingStore: { getFieldMeta: () => undefined },
        bindingMeta: { maxHeight: 12 },
      })
    ).toEqual({ maxHeight: 12 });

    expect(
      resolveImageFieldLimitsFromSources({
        bindingProp: 'Photo',
        propsStore: {},
        bindingMeta: { maxWidth: 7 },
      })
    ).toEqual({ maxWidth: 7 });

    // Both props falsy so `bindingProp || propsProp || ''` takes the final `''` branch.
    expect(
      resolveImageFieldLimitsFromSources({
        bindingProp: undefined,
        propsProp: '',
        bindingStore: {
          getFieldMeta: () => ({ maxUploadBytes: 999 }),
        },
        bindingMeta: { maxHeight: 4 },
      })
    ).toEqual({ maxHeight: 4 });

    // Whitespace-only leaf after trim must also skip store lookup.
    expect(
      resolveImageFieldLimitsFromSources({
        bindingProp: '   ',
        propsProp: null,
        bindingStore: {
          getFieldMeta: () => ({ maxWidth: 1 }),
        },
        bindingMeta: { maxHeight: 5 },
      })
    ).toEqual({ maxHeight: 5 });
  });

  it('imageFieldLimitErrorMessage covers all reasons', () => {
    expect(imageFieldLimitErrorMessage({ ok: false, reason: 'fileTooLarge', detail: '1 KB' })).toMatch(/1 KB|上限/);
    expect(imageFieldLimitErrorMessage({ ok: false, reason: 'widthTooLarge', detail: '10' })).toMatch(/10/);
    expect(imageFieldLimitErrorMessage({ ok: false, reason: 'heightTooLarge', detail: '20' })).toMatch(/20/);
  });

  it('reportImageFieldValidation invokes onError on failure and returns true on success', async () => {
    const onError = vi.fn();
    expect(await reportImageFieldValidation(makeFile(2048), limits, onError, async () => ({ width: 1, height: 1 }))).toBe(false);
    expect(onError).toHaveBeenCalled();

    onError.mockClear();
    expect(await reportImageFieldValidation(makeFile(10), limits, onError, async () => ({ width: 1, height: 1 }))).toBe(true);
    expect(onError).not.toHaveBeenCalled();
  });

  describe('readImageNaturalDimensions', () => {
    const originalCreateImageBitmap = globalThis.createImageBitmap;
    const originalImage = globalThis.Image;
    const originalURL = globalThis.URL;

    afterEach(() => {
      if (originalCreateImageBitmap) {
        (globalThis as any).createImageBitmap = originalCreateImageBitmap;
      } else {
        delete (globalThis as any).createImageBitmap;
      }
      (globalThis as any).Image = originalImage;
      (globalThis as any).URL = originalURL;
    });

    it('uses createImageBitmap when available', async () => {
      const close = vi.fn();
      (globalThis as any).createImageBitmap = vi.fn(async () => ({ width: 11, height: 22, close }));
      await expect(readImageNaturalDimensions(makeFile(8))).resolves.toEqual({ width: 11, height: 22 });
      expect(close).toHaveBeenCalled();

      (globalThis as any).createImageBitmap = vi.fn(async () => ({ width: 3, height: 4 }));
      await expect(readImageNaturalDimensions(makeFile(8))).resolves.toEqual({ width: 3, height: 4 });
    });

    it('returns undefined when createImageBitmap throws', async () => {
      (globalThis as any).createImageBitmap = vi.fn(async () => {
        throw new Error('bad image');
      });
      await expect(readImageNaturalDimensions(makeFile(8))).resolves.toBeUndefined();
    });

    it('falls back to Image onload / onerror / missing URL helpers', async () => {
      delete (globalThis as any).createImageBitmap;

      class FakeImage {
        onload: (() => void) | null = null;
        onerror: (() => void) | null = null;
        naturalWidth = 33;
        naturalHeight = 44;
        set src(_v: string) {
          queueMicrotask(() => this.onload?.());
        }
      }
      (globalThis as any).Image = FakeImage;
      const createObjectURL = vi.fn(() => 'blob:test');
      const revokeObjectURL = vi.fn();
      (globalThis as any).URL = { createObjectURL, revokeObjectURL };

      await expect(readImageNaturalDimensions(makeFile(8))).resolves.toEqual({ width: 33, height: 44 });
      expect(revokeObjectURL).toHaveBeenCalledWith('blob:test');

      class ErrImage {
        onload: (() => void) | null = null;
        onerror: (() => void) | null = null;
        set src(_v: string) {
          queueMicrotask(() => this.onerror?.());
        }
      }
      (globalThis as any).Image = ErrImage;
      await expect(readImageNaturalDimensions(makeFile(8))).resolves.toBeUndefined();

      (globalThis as any).URL = { createObjectURL: undefined, revokeObjectURL };
      await expect(readImageNaturalDimensions(makeFile(8))).resolves.toBeUndefined();

      (globalThis as any).Image = FakeImage;
      (globalThis as any).URL = { createObjectURL: vi.fn(() => 'blob:norevoke') };
      await expect(readImageNaturalDimensions(makeFile(8))).resolves.toEqual({ width: 33, height: 44 });

      delete (globalThis as any).URL;
      await expect(readImageNaturalDimensions(makeFile(8))).resolves.toBeUndefined();

      delete (globalThis as any).Image;
      await expect(readImageNaturalDimensions(makeFile(8))).resolves.toBeUndefined();
    });
  });
});

describe('OImageField upload limit gate (PR-P2-F3)', () => {
  beforeEach(async () => {
    const { ElMessage } = await import('element-plus');
    vi.mocked(ElMessage.error).mockClear();
    vi.restoreAllMocks();
  });

  async function mountField(binding: UseField) {
    return mount(OImageField as any, {
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
          Picture: true,
          UploadFilled: true,
        },
      },
    });
  }

  it('rejects oversized upload via ElMessage and keeps value empty', async () => {
    const limitsMod = await import('./imageFieldLimits');
    vi.spyOn(limitsMod, 'reportImageFieldValidation').mockImplementation(async (_file, _limits, onError) => {
      onError('Image exceeds maximum size (1 KB)');
      return false;
    });

    const binding = makeBinding({
      meta: { maxUploadBytes: 100, maxWidth: 50, maxHeight: 50 },
      store: {
        getFieldMeta: () => ({ maxUploadBytes: 100, maxWidth: 50, maxHeight: 50 }),
      },
    });
    const wrapper = await mountField(binding);
    const upload = wrapper.findComponent({ name: 'ElUploadStub' });
    const onChange = upload.props('onChange') as (file: any) => Promise<void>;
    await onChange({ raw: makeFile(200), name: 'photo.png', status: 'ready' });
    await flushPromises();

    const { ElMessage } = await import('element-plus');
    expect(ElMessage.error).toHaveBeenCalledWith('Image exceeds maximum size (1 KB)');
    expect(binding.fieldRef().value).toBeNull();
  });

  it('rejects oversized width and height using binding meta fallback', async () => {
    const limitsMod = await import('./imageFieldLimits');
    const spy = vi.spyOn(limitsMod, 'reportImageFieldValidation');
    spy.mockImplementationOnce(async (_file, _limits, onError) => {
      onError('Image width exceeds maximum (40 px)');
      return false;
    });

    const binding = makeBinding({
      meta: { maxUploadBytes: 10_000, maxWidth: 40, maxHeight: 30 },
    });
    const wrapper = await mountField(binding);
    const upload = wrapper.findComponent({ name: 'ElUploadStub' });
    const onChange = upload.props('onChange') as (file: any) => Promise<void>;

    await onChange({ raw: makeFile(20), name: 'photo.png', status: 'ready' });
    await flushPromises();
    const { ElMessage } = await import('element-plus');
    expect(ElMessage.error).toHaveBeenCalledWith('Image width exceeds maximum (40 px)');

    vi.mocked(ElMessage.error).mockClear();
    spy.mockImplementationOnce(async (_file, _limits, onError) => {
      onError('Image height exceeds maximum (30 px)');
      return false;
    });
    await onChange({ raw: makeFile(20), name: 'photo.png', status: 'ready' });
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('Image height exceeds maximum (30 px)');
  });

  it('accepts valid image and writes pending attachment value', async () => {
    const limitsMod = await import('./imageFieldLimits');
    vi.spyOn(limitsMod, 'reportImageFieldValidation').mockResolvedValue(true);

    const binding = makeBinding({
      meta: { maxUploadBytes: 10_000, maxWidth: 200, maxHeight: 200 },
      store: {
        getFieldMeta: (name: string) =>
          name === 'Photo' ? { maxUploadBytes: 10_000, maxWidth: 200, maxHeight: 200 } : undefined,
      },
    });

    const wrapper = await mountField(binding);
    const upload = wrapper.findComponent({ name: 'ElUploadStub' });
    const onChange = upload.props('onChange') as (file: any) => Promise<void>;
    await onChange({ raw: makeFile(32), name: 'photo.png', status: 'ready' });
    await flushPromises();

    const { ElMessage } = await import('element-plus');
    expect(ElMessage.error).not.toHaveBeenCalled();
    expect(binding.fieldRef().value).toMatchObject({
      kind: 'set',
      fileName: 'photo.png',
    });
  });
});
