// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { navigatePartnerDetail, resolvePartnerDetailPath } from './partner_list_nav';

test('resolvePartnerDetailPath: builds path from a raw row when allowed', () => {
  expect(resolvePartnerDetailPath({ Id: '  ptn-1  ' }, true)).toBe('/partner/partners/ptn-1');
});

test('resolvePartnerDetailPath: unwraps { row } payloads', () => {
  expect(resolvePartnerDetailPath({ row: { Id: 'ptn-9' }, index: 0 }, true)).toBe(
    '/partner/partners/ptn-9',
  );
});

test('resolvePartnerDetailPath: denies or blank id fail closed', () => {
  expect(resolvePartnerDetailPath({ Id: 'ptn-1' }, false)).toBeNull();
  expect(resolvePartnerDetailPath({ Id: '   ' }, true)).toBeNull();
  expect(resolvePartnerDetailPath({}, true)).toBeNull();
  expect(resolvePartnerDetailPath(null, true)).toBeNull();
});

test('navigatePartnerDetail: pushes when allowed and skips otherwise', () => {
  const pushes: string[] = [];
  const push = (path: string) => {
    pushes.push(path);
  };
  expect(navigatePartnerDetail({ Id: 'ptn-2' }, true, push)).toBe(true);
  expect(pushes).toEqual(['/partner/partners/ptn-2']);
  expect(navigatePartnerDetail({ Id: 'ptn-2' }, false, push)).toBe(false);
  expect(navigatePartnerDetail({ Id: '' }, true, push)).toBe(false);
  expect(pushes).toEqual(['/partner/partners/ptn-2']);
});
