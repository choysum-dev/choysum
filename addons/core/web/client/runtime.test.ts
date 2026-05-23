// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import * as runtimeApi from './runtime';

describe('core/web client runtime entrypoint export surface', () => {
  it('keeps advanced runtime helpers behind an explicit sub-entrypoint', () => {
    expect(Object.keys(runtimeApi).sort()).toEqual(['createApiRuntime', 'createApiStateScope', 'initializeApiRuntime', 'useApiState']);
  });
});
