// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '@/core/service/orm/model/model';
import { Field } from '@/core/service/orm/decorator/field';
import { Model } from '@/core/service/orm/decorator/model';
import { MetadataStorage } from '@/core/service/orm/metadata/storage';
import { ChoysumError } from '@/core/service/error';
import { validateAttachmentContentFieldLimits } from '../models/_attachment_binding_ops';

@Model('BindLimitPilot', { application: 'demo' })
class BindLimitPilot extends BaseModel {
  @Field({ type: 'image', maxUploadBytes: 100, maxWidth: 800, maxHeight: 600 } as any)
  Photo!: string | null;
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
});
