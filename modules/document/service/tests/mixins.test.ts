// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';
import AttachmentOwnerMixin from '../mixins/attachment_owner_model';
import AttachmentBinding from '../models/attachment_binding';

/**
 * Harness consumer for AttachmentOwnerMixin extend contract (not a persisted domain model).
 */
class AttachmentOwnerHarness extends AttachmentOwnerMixin {}

test('AttachmentOwnerMixin: AttachmentBinding and harness extend the mixin', () => {
  expect(Object.prototype.isPrototypeOf.call(AttachmentOwnerMixin, AttachmentBinding)).toBe(false);
  expect(Object.prototype.isPrototypeOf.call(BaseModel, AttachmentBinding)).toBe(true);
  expect(Object.prototype.isPrototypeOf.call(AttachmentOwnerMixin, AttachmentOwnerHarness)).toBe(true);
});

test('AttachmentOwnerMixin: harness exposes bind/unbind entry points', () => {
  expect(typeof AttachmentOwnerHarness.AttachmentBind).toBe('function');
  expect(typeof AttachmentOwnerHarness.AttachmentUnbind).toBe('function');
});
