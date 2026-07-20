// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { createTranslate } from '@/web/web/i18n';
import { metaMenus } from '../menu/menus';
import { metaRoutes } from '../route/routes';

const rootTitle = createTranslate('meta', { scope: 'web/menu/menus' })._lt('Module Management');
const boardTitle = createTranslate('meta', { scope: 'web/route/routes' })._lt('Module Management');

describe('meta menu/route _lt titles', () => {
  it('pins TermReference titles on meta menus', () => {
    const root = metaMenus[0] as any;
    expect(root.title).toBe('Module Management');
    expect(root.titleText).toEqual(rootTitle);
  });

  it('pins TermReference titles on meta routes', () => {
    const board = metaRoutes.find((route: any) => route.name === 'MetaModuleList') as any;
    expect(board).toBeTruthy();
    expect(board.meta?.pageTitle).toBe('Module Management');
    expect(board.meta?.pageTitleText).toEqual(boardTitle);
  });
});
