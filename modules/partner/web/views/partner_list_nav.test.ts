// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { navigatePartnerDetail, resolvePartnerDetailPath } from './partner_list_nav';
import type { RowEventPayload } from '@/web';

function rowPayload(id: unknown): RowEventPayload {
  return {
    row: { Id: id } as RowEventPayload['row'],
    rowIndex: 0,
    rowKey: undefined,
    event: new Event('click'),
  };
}

test('resolvePartnerDetailPath: builds path when allowed', () => {
  expect(resolvePartnerDetailPath(rowPayload('  ptn-1  '), true)).toBe('/partner/partners/ptn-1');
});

test('resolvePartnerDetailPath: denies or blank id fail closed', () => {
  expect(resolvePartnerDetailPath(rowPayload('ptn-1'), false)).toBeNull();
  expect(resolvePartnerDetailPath(rowPayload('   '), true)).toBeNull();
  expect(resolvePartnerDetailPath(rowPayload(undefined), true)).toBeNull();
});

test('navigatePartnerDetail: pushes when allowed and skips otherwise', () => {
  const pushed: string[] = [];
  const push = (path: string) => {
    pushed.push(path);
  };
  expect(navigatePartnerDetail(rowPayload('ptn-2'), true, push)).toBe(true);
  expect(pushed).toEqual(['/partner/partners/ptn-2']);
  expect(navigatePartnerDetail(rowPayload('ptn-2'), false, push)).toBe(false);
  expect(navigatePartnerDetail(rowPayload(''), true, push)).toBe(false);
});
