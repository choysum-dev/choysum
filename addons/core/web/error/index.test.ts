// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import * as webError from './index';

describe('core/web/error entrypoint export surface', () => {
  it('stays explicit and stable', () => {
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

  it('runtime exports remain callable', () => {
    expect(typeof webError.errorAction).toBe('function');
    expect(typeof webError.errorMessageKey).toBe('function');
    expect(typeof webError.toUIErrorState).toBe('function');
  });
});
