// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as webError from './index';

test('core/web/error entrypoint export surface: stays explicit and stable', () => {
  expect(Object.keys(webError).sort()).toEqual([
    'ChoysumError',
    'ErrorFactory',
    'ErrorInfoSchema',
    'GrpcCode',
    'createDomainErrorHandlers',
    'errorAction',
    'errorAs',
    'errorMessageKey',
    'generateErrorId',
    'isErrorOf',
    'toUIErrorState',
    'validateErrorCode',
  ]);
});

test('core/web/error entrypoint export surface: runtime exports remain callable', () => {
  expect(typeof webError.errorAction).toBe('function');
  expect(typeof webError.errorMessageKey).toBe('function');
  expect(typeof webError.toUIErrorState).toBe('function');
});
