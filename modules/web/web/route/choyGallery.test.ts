// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { fnRecorder } from '@/web/web/__tests__/mountApp';
import { choyUiRoutes, registerChoyGalleryRoute, setupRouter } from './choyGallery';

test('setupRouter > skips when router is missing', () => {
  const warn = fnRecorder();
  const prev = console.warn;
  console.warn = warn as any;
  try {
    setupRouter({} as any);
    setupRouter({ router: null } as any);
  } finally {
    console.warn = prev;
  }
  expect(warn.calls.length).toBe(2);
});

test('setupRouter > skips when AppLayout is not registered', () => {
  const warn = fnRecorder();
  const hasRoute = fnRecorder((name: string) => name !== 'AppLayout');
  const addRoute = fnRecorder();
  const prev = console.warn;
  console.warn = warn as any;
  try {
    setupRouter({ router: { hasRoute, addRoute } } as any);
  } finally {
    console.warn = prev;
  }
  expect(warn.calls.length).toBe(1);
  expect(addRoute.calls.length).toBe(0);
});

test('setupRouter > adds missing gallery routes under AppLayout', () => {
  const existing = new Set<string>(['AppLayout', String(choyUiRoutes[0]?.name ?? '')]);
  const hasRoute = fnRecorder((name: string) => existing.has(String(name)));
  const addRoute = fnRecorder((_parent: string, route: { name?: string }) => {
    if (route?.name != null) {
      existing.add(String(route.name));
    }
  });
  setupRouter({ router: { hasRoute, addRoute } } as any);
  const addedNames = addRoute.calls.map(call => String((call[1] as { name?: string })?.name ?? ''));
  expect(addedNames).not.toContain(String(choyUiRoutes[0]?.name ?? ''));
  expect(addedNames.length).toBe(choyUiRoutes.length - 1);
});

test('registerChoyGalleryRoute > delegates to setupRouter', () => {
  const addRoute = fnRecorder();
  const hasRoute = fnRecorder((name: string) => name === 'AppLayout');
  registerChoyGalleryRoute({ router: { hasRoute, addRoute } } as any);
  expect(addRoute.calls.length).toBe(choyUiRoutes.length);
});
