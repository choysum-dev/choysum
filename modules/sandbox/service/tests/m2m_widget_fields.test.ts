// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import SandboxRecord from '@/sandbox/service/models/sandbox_record';
import SandboxRecordCategory from '@/sandbox/service/models/sandbox_record_category';
import SandboxRecordTag from '@/sandbox/service/models/sandbox_record_tag';
import SandboxRecordTagChip from '@/sandbox/service/models/sandbox_record_tag_chip';
import { MetadataStorage } from '@/core/service/api/metadata';

test('sandbox.SandboxRecord TagChips is ManyToMany targeting SandboxTag', () => {
  const meta = MetadataStorage.instance.getModelMetadata(SandboxRecord as any);
  const field = meta?.fields?.get('TagChips');
  expect(field?.type).toBe('ManyToMany');
  const target = String((field as any)?.relation?.targetModel ?? (field as any)?.relationModel ?? '');
  expect(target).toMatch(/SandboxTag/);
});

test('sandbox.SandboxRecord Categories is ManyToMany targeting SandboxCategory', () => {
  const meta = MetadataStorage.instance.getModelMetadata(SandboxRecord as any);
  const field = meta?.fields?.get('Categories');
  expect(field?.type).toBe('ManyToMany');
  const target = String((field as any)?.relation?.targetModel ?? (field as any)?.relationModel ?? '');
  expect(target).toMatch(/SandboxCategory/);
  expect(field?.condition).toEqual(['IsActive', '=', true]);
});

test('sandbox.SandboxRecord TagTableRefIds is ManyToManyRef targeting SandboxTag', () => {
  const meta = MetadataStorage.instance.getModelMetadata(SandboxRecord as any);
  const field = meta?.fields?.get('TagTableRefIds');
  expect(field?.type).toBe('ManyToManyRef');
  const target = String((field as any)?.relation?.targetModel ?? (field as any)?.relationModel ?? '');
  expect(target).toMatch(/SandboxTag/);
});

test('sandbox M2M join models disable soft delete', () => {
  for (const ctor of [SandboxRecordTag, SandboxRecordTagChip, SandboxRecordCategory]) {
    const meta = MetadataStorage.instance.getModelMetadata(ctor as any);
    expect(meta?.softDelete).toBe(false);
  }
});
