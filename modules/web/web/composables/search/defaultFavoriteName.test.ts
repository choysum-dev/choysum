// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { createTermReference } from '@/core/service/i18n';
import {
  modelIdentityFromStore,
  pickDefaultFavoriteName,
  routeTitleFromLocation,
  stableTitleSource,
} from './defaultFavoriteName';

describe('pickDefaultFavoriteName', () => {
  it('prefers breadcrumb tip over route, menu, and model identity', () => {
    expect(
      pickDefaultFavoriteName({
        breadcrumbTip: ' Users ',
        routeTitle: 'Route',
        menuTitle: 'Menu',
        modelIdentity: 'auth.User',
      })
    ).toBe('Users');
  });

  it('falls back through route, menu, then model identity', () => {
    expect(pickDefaultFavoriteName({ routeTitle: 'Route', menuTitle: 'Menu' })).toBe('Route');
    expect(pickDefaultFavoriteName({ menuTitle: 'Menu', modelIdentity: 'auth.User' })).toBe('Menu');
    expect(pickDefaultFavoriteName({ modelIdentity: 'auth.User' })).toBe('auth.User');
    expect(pickDefaultFavoriteName({})).toBe('');
  });

  it('skips blank strings', () => {
    expect(pickDefaultFavoriteName({ breadcrumbTip: '  ', routeTitle: '', menuTitle: 'Ok' })).toBe('Ok');
  });
});

describe('stableTitleSource', () => {
  it('uses TermReference.src and never requires a live translator', () => {
    const term = createTermReference('web', 'Users', { scope: 'web/pages' });
    expect(stableTitleSource('fallback', term)).toBe('Users');
    expect(stableTitleSource('Plain')).toBe('Plain');
  });
});

describe('routeTitleFromLocation', () => {
  it('reads pageTitle string and function', () => {
    expect(routeTitleFromLocation({ meta: { pageTitle: 'Contacts' } })).toBe('Contacts');
    expect(
      routeTitleFromLocation({
        meta: { pageTitle: (r: any) => `Hi ${r.name}` },
        name: 'World',
      })
    ).toBe('Hi World');
  });

  it('falls back to meta.title and ignores technical route.name', () => {
    expect(routeTitleFromLocation({ meta: { title: 'Legacy' }, name: 'auth.users' })).toBe('Legacy');
    expect(routeTitleFromLocation({ name: 'auth.users' })).toBe('');
  });

  it('uses pageTitleText src (not translated UI string)', () => {
    expect(
      routeTitleFromLocation({
        meta: {
          pageTitleText: createTermReference('web', 'Users', { scope: 'web/pages' }),
        },
      })
    ).toBe('Users');
  });
});

describe('modelIdentityFromStore', () => {
  it('joins application and modelName', () => {
    expect(modelIdentityFromStore({ application: 'auth', modelName: 'User' })).toBe('auth.User');
    expect(modelIdentityFromStore({ application: 'auth' })).toBe('auth');
    expect(modelIdentityFromStore(null)).toBe('');
  });
});
