// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  resolveUserMenuAvatarDisplaySrc,
  resolveUserMenuAvatarPreviewUrl,
} from './user_menu_avatar';

test('resolveUserMenuAvatarPreviewUrl: empty / non-attachment → undefined', () => {
  expect(resolveUserMenuAvatarPreviewUrl(null)).toBeUndefined();
  expect(resolveUserMenuAvatarPreviewUrl(undefined)).toBeUndefined();
  expect(resolveUserMenuAvatarPreviewUrl('')).toBeUndefined();
  expect(resolveUserMenuAvatarPreviewUrl({ kind: 'clear' })).toBeUndefined();
});

test('resolveUserMenuAvatarPreviewUrl: prefers previewUrl', () => {
  expect(
    resolveUserMenuAvatarPreviewUrl({
      kind: 'set',
      previewUrl: ' /pv/a.png ',
      downloadUrl: '/dl/a.png',
    }),
  ).toBe('/pv/a.png');
});

test('resolveUserMenuAvatarPreviewUrl: bare Browse binding id → content URL', () => {
  expect(resolveUserMenuAvatarPreviewUrl('bind-avatar-1')).toBe(
    '/_document/bindings/bind-avatar-1/content',
  );
});

test('resolveUserMenuAvatarPreviewUrl: attachment object with binding id only', () => {
  expect(resolveUserMenuAvatarPreviewUrl({ attachmentBindingId: 'bind-2' })).toBe(
    '/_document/bindings/bind-2/content',
  );
});

test('resolveUserMenuAvatarDisplaySrc: bare binding id auth-fetches content', async () => {
  const calls: string[] = [];
  const src = await resolveUserMenuAvatarDisplaySrc('bind-avatar-1', async url => {
    calls.push(url);
    return 'data:image/png;base64,from-id';
  });
  expect(calls).toEqual(['/_document/bindings/bind-avatar-1/content']);
  expect(src).toBe('data:image/png;base64,from-id');
});

test('resolveUserMenuAvatarDisplaySrc: passthrough non-document URLs', async () => {
  const fetchAuthed = async () => {
    throw new Error('should not fetch');
  };
  expect(await resolveUserMenuAvatarDisplaySrc({ kind: 'set', previewUrl: 'https://cdn/x.png' }, fetchAuthed)).toBe(
    'https://cdn/x.png',
  );
  expect(await resolveUserMenuAvatarDisplaySrc({ kind: 'set', previewUrl: 'data:image/png;base64,aa' }, fetchAuthed)).toBe(
    'data:image/png;base64,aa',
  );
});

test('resolveUserMenuAvatarDisplaySrc: auth-fetches internal binding content', async () => {
  const calls: string[] = [];
  const src = await resolveUserMenuAvatarDisplaySrc(
    { kind: 'set', previewUrl: '/_document/bindings/b1/content' },
    async url => {
      calls.push(url);
      return 'data:image/png;base64,auth';
    },
  );
  expect(calls).toEqual(['/_document/bindings/b1/content']);
  expect(src).toBe('data:image/png;base64,auth');
});

test('resolveUserMenuAvatarDisplaySrc: auth fetch miss → undefined', async () => {
  const src = await resolveUserMenuAvatarDisplaySrc(
    { kind: 'set', previewUrl: '/_document/bindings/b1/content' },
    async () => undefined,
  );
  expect(src).toBeUndefined();
});
