// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveCollaborationCaps } from './collaborationCaps';

test('resolveCollaborationCaps: empty registry → all false', () => {
  expect(resolveCollaborationCaps([])).toEqual({
    message: false,
    audit: false,
    document: false,
  });
});

test('resolveCollaborationCaps: message models alone → message true', () => {
  expect(
    resolveCollaborationCaps([
      'message.Message',
      'message.Notification',
      'message.Follower',
      'other.Model',
    ])
  ).toEqual({
    message: true,
    audit: false,
    document: false,
  });
});

test('resolveCollaborationCaps: partial message models → message false', () => {
  expect(resolveCollaborationCaps(['message.Message', 'message.Notification'])).toEqual({
    message: false,
    audit: false,
    document: false,
  });
});

test('resolveCollaborationCaps: full set → all true', () => {
  expect(
    resolveCollaborationCaps([
      'message.Message',
      'message.Notification',
      'message.Follower',
      'audit.FieldChange',
      'document.AttachmentBinding',
    ])
  ).toEqual({
    message: true,
    audit: true,
    document: true,
  });
});
