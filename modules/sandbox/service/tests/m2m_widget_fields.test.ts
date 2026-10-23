// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import SandboxRecord from '@/sandbox/service/models/sandbox_record';
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
});
