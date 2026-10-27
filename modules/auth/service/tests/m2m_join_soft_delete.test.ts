// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '@/core/service/api/metadata';
import RoleInheritance from '@/auth/service/models/role_inheritance';
import UserRole from '@/auth/service/models/user_role';

test('RoleInheritance pure-link join disables soft delete', () => {
  const meta = MetadataStorage.instance.getModelMetadata(RoleInheritance as any);
  expect(meta?.softDelete).toBe(false);
});

test('UserRole association join keeps soft delete enabled', () => {
  const meta = MetadataStorage.instance.getModelMetadata(UserRole as any);
  expect(meta?.softDelete).not.toBe(false);
});
