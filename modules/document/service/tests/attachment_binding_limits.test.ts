// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '@/core/service/orm/model/model';
import { Field } from '@/core/service/orm/decorator/field';
import { Model } from '@/core/service/orm/decorator/model';
import { MetadataStorage } from '@/core/service/orm/metadata/storage';
import { ChoysumError } from '@/core/service/error';
import { DEFAULT_GLOBAL_MAX_UPLOAD_BYTES } from '@/core/service/orm/upload_limits';
import { validateAttachmentContentFieldLimits } from '../models/_attachment_binding_ops';

@Model('BindLimitPilot', { application: 'demo' })
class BindLimitPilot extends BaseModel {
  @Field({ type: 'image', maxUploadBytes: 100, maxWidth: 800, maxHeight: 600 } as any)
  Photo!: string | null;

  @Field({ type: 'image' } as any)
  Plain!: string | null;

  @Field({ type: 'binary', maxUploadBytes: 50 } as any)
  Payload!: string | null;

  @Field({ type: 'image', maxWidth: 100 } as any)
  WidthOnly!: string | null;

  @Field({ type: 'image', maxHeight: 100 } as any)
  HeightOnly!: string | null;
}

function expectChoysumError(fn: () => void): ChoysumError {
  try {
    fn();
    throw new Error('expected ChoysumError');
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    return err as ChoysumError;
  }
}

test('validateAttachmentContentFieldLimits rejects oversized bytes on image field (PR-P2-F3)', () => {
  expect(MetadataStorage.instance.getModelMetadata(BindLimitPilot as any).fields.get('Photo')?.maxUploadBytes).toBe(100);

  const err = expectChoysumError(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'Photo', {
      SizeBytes: 200,
    } as any)
  );
  expect(String(err.message || '')).toContain('maxUploadBytes');
});

test('validateAttachmentContentFieldLimits rejects oversized dimensions when probe present', () => {
  const widthErr = expectChoysumError(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'Photo', {
      SizeBytes: 50,
      ImageWidth: 900,
      ImageHeight: 400,
    } as any)
  );
  expect(String(widthErr.message || '')).toContain('maxWidth');

  const heightErr = expectChoysumError(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'Photo', {
      SizeBytes: 50,
      ImageWidth: 400,
      ImageHeight: 700,
    } as any)
  );
  expect(String(heightErr.message || '')).toContain('maxHeight');
});

test('validateAttachmentContentFieldLimits skips dimension check when probe missing', () => {
  expect(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'Photo', {
      SizeBytes: 50,
    } as any)
  ).not.toThrow();
});

test('validateAttachmentContentFieldLimits no-op for fields without limits', () => {
  expect(() =>
    validateAttachmentContentFieldLimits('auth.User', 'Avatar', {
      SizeBytes: 50 * 1024 * 1024,
    } as any)
  ).not.toThrow();

  expect(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'Plain', {
      SizeBytes: 50 * 1024 * 1024,
      ImageWidth: 9999,
      ImageHeight: 9999,
    } as any)
  ).not.toThrow();
});

test('validateAttachmentContentFieldLimits no-op for unknown model or missing field', () => {
  expect(() =>
    validateAttachmentContentFieldLimits('demo.__NoSuchModel__', 'Photo', {
      SizeBytes: 200,
    } as any)
  ).not.toThrow();

  expect(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'MissingField', {
      SizeBytes: 200,
    } as any)
  ).not.toThrow();
});

test('validateAttachmentContentFieldLimits enforces binary byte caps and ignores non-finite sizes', () => {
  const err = expectChoysumError(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'Payload', {
      SizeBytes: 80,
    } as any)
  );
  expect(String(err.message || '')).toContain('maxUploadBytes');

  expect(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'Payload', {
      SizeBytes: Number.NaN,
    } as any)
  ).not.toThrow();

  expect(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'Payload', {
      SizeBytes: 10,
    } as any)
  ).not.toThrow();
});

test('validateAttachmentContentFieldLimits applies global byte cap via Math.min', () => {
  const photo = MetadataStorage.instance.getModelMetadata(BindLimitPilot as any).fields.get('Photo')!;
  const previous = photo.maxUploadBytes;
  photo.maxUploadBytes = DEFAULT_GLOBAL_MAX_UPLOAD_BYTES + 1024;
  try {
    const err = expectChoysumError(() =>
      validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'Photo', {
        SizeBytes: DEFAULT_GLOBAL_MAX_UPLOAD_BYTES + 1,
      } as any)
    );
    expect(String(err.message || '')).toContain('maxUploadBytes');
    expect(err.metadata.maxUploadBytes).toBe(String(DEFAULT_GLOBAL_MAX_UPLOAD_BYTES));
  } finally {
    photo.maxUploadBytes = previous;
  }
});

test('validateAttachmentContentFieldLimits supports width-only and height-only limits', () => {
  expectChoysumError(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'WidthOnly', {
      SizeBytes: 1,
      ImageWidth: 200,
    } as any)
  );

  expect(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'WidthOnly', {
      SizeBytes: 1,
      ImageWidth: 50,
      ImageHeight: 9999,
    } as any)
  ).not.toThrow();

  expectChoysumError(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'HeightOnly', {
      SizeBytes: 1,
      ImageHeight: 200,
    } as any)
  );

  expect(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'HeightOnly', {
      SizeBytes: 1,
      ImageWidth: 9999,
      ImageHeight: 50,
    } as any)
  ).not.toThrow();
});

test('validateAttachmentContentFieldLimits accepts in-range image with probe', () => {
  expect(() =>
    validateAttachmentContentFieldLimits('demo.BindLimitPilot', 'Photo', {
      SizeBytes: 50,
      ImageWidth: 100,
      ImageHeight: 100,
    } as any)
  ).not.toThrow();
});
