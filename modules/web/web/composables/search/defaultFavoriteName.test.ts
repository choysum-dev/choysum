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

  it('falls back to title when TermReference.src is empty', () => {
    const term = createTermReference('web', '', { scope: 'web/pages' });
    expect(stableTitleSource('Fallback', term)).toBe('Fallback');
  });

  it('returns empty when TermReference.src and title are both empty', () => {
    const term = createTermReference('web', '', { scope: 'web/pages' });
    expect(stableTitleSource('', term)).toBe('');
    expect(stableTitleSource(undefined, term)).toBe('');
  });

  it('ignores non-TermReference titleText', () => {
    expect(stableTitleSource('Plain', { not: 'a term' } as any)).toBe('Plain');
    expect(stableTitleSource(undefined)).toBe('');
  });
});

describe('routeTitleFromLocation', () => {
  it('returns empty when route is missing', () => {
    expect(routeTitleFromLocation(null)).toBe('');
    expect(routeTitleFromLocation(undefined)).toBe('');
  });

  it('treats missing meta as empty object', () => {
    expect(routeTitleFromLocation({})).toBe('');
    expect(routeTitleFromLocation({ meta: null })).toBe('');
  });

  it('reads pageTitle string and function', () => {
    expect(routeTitleFromLocation({ meta: { pageTitle: 'Contacts' } })).toBe('Contacts');
    expect(
      routeTitleFromLocation({
        meta: { pageTitle: (r: any) => `Hi ${r.name}` },
        name: 'World',
      })
    ).toBe('Hi World');
  });

  it('treats falsy pageTitle function results as empty', () => {
    expect(routeTitleFromLocation({ meta: { pageTitle: () => null } })).toBe('');
    expect(routeTitleFromLocation({ meta: { pageTitle: () => undefined } })).toBe('');
    expect(routeTitleFromLocation({ meta: { pageTitle: () => '  ' } })).toBe('');
  });

  it('swallows pageTitle function errors', () => {
    expect(
      routeTitleFromLocation({
        meta: {
          pageTitle: () => {
            throw new Error('boom');
          },
        },
      })
    ).toBe('');
  });

  it('skips blank pageTitle and falls back to meta.title', () => {
    expect(routeTitleFromLocation({ meta: { pageTitle: '  ', title: 'Legacy' } })).toBe('Legacy');
    expect(routeTitleFromLocation({ meta: { title: '  ' } })).toBe('');
    expect(routeTitleFromLocation({ meta: { title: null } })).toBe('');
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
    expect(modelIdentityFromStore({ modelName: 'User' })).toBe('User');
    expect(modelIdentityFromStore({ application: '  ', modelName: '  ' })).toBe('');
    expect(modelIdentityFromStore({ application: null, modelName: null })).toBe('');
    expect(modelIdentityFromStore(null)).toBe('');
    expect(modelIdentityFromStore(undefined)).toBe('');
  });
});

describe('pickDefaultFavoriteName nullish sources', () => {
  it('skips null and undefined entries in the preference chain', () => {
    expect(
      pickDefaultFavoriteName({
        breadcrumbTip: null,
        routeTitle: undefined,
        menuTitle: null,
        modelIdentity: 'demo.Widget',
      })
    ).toBe('demo.Widget');
  });
});
