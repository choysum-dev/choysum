// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import * as trackingApi from './index';

describe('core/web tracking entrypoint export surface', () => {
  it('keeps the tracking facade limited to draft tracking primitives', () => {
    expect(Object.keys(trackingApi).sort()).toEqual(['TrackedModel', 'track']);
  });
});
