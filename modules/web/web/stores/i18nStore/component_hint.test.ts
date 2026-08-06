// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { componentHintFromScope } from './component_hint';

describe('componentHintFromScope', () => {
  it('strips @field suffix for page scopes', () => {
    expect(componentHintFromScope('web/pages/Login@title')).toBe('web/pages/Login');
    expect(componentHintFromScope('game.rescue')).toBe('game.rescue');
    expect(componentHintFromScope('')).toBe('');
  });
});
