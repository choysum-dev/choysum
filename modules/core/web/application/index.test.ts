// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import * as applicationApi from './index';

describe('core/web application entrypoint export surface', () => {
  it('keeps the runtime facade limited to app container primitives', () => {
    expect(Object.keys(applicationApi).sort()).toEqual(['createApp', 'getPlugins']);
  });
});
