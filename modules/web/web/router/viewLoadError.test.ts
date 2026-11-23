// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  clearViewLoadError,
  isRouteChunkLoadError,
  setViewLoadError,
  viewLoadError,
} from './viewLoadError';

afterEach(() => {
  clearViewLoadError();
});

test('isRouteChunkLoadError: recognizes common chunk failure messages', () => {
  expect(
    isRouteChunkLoadError(new Error('Failed to fetch dynamically imported module: /web/assets/Company-x.js'))
  ).toBe(true);
  expect(isRouteChunkLoadError(new Error('Loading chunk 5 failed'))).toBe(true);
  expect(isRouteChunkLoadError(new Error('net::ERR_CONNECTION_REFUSED'))).toBe(true);
  expect(isRouteChunkLoadError(new Error('InvalidArgument: bad field'))).toBe(false);
});

test('viewLoadError: set and clear', () => {
  expect(viewLoadError.value).toBeNull();
  setViewLoadError({ message: 'x', fullPath: '/web/companies' });
  expect(viewLoadError.value?.fullPath).toBe('/web/companies');
  clearViewLoadError();
  expect(viewLoadError.value).toBeNull();
});
