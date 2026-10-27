// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '@/core/service/api/metadata';
import MetaModuleDependency from '@/meta/service/models/module_dependency';

test('MetaModuleDependency registry join disables soft delete', () => {
  const meta = MetadataStorage.instance.getModelMetadata(MetaModuleDependency as any);
  expect(meta?.softDelete).toBe(false);
});
