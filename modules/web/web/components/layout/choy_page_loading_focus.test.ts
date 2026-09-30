// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { blurFocusedDescendant } from './choy_page_loading_focus';

describe('blurFocusedDescendant', () => {
  test('blurs activeElement when it is inside root', () => {
    const root = document.createElement('div');
    const btn = document.createElement('button');
    let blurred = false;
    btn.blur = () => {
      blurred = true;
    };
    root.appendChild(btn);
    document.body.appendChild(root);

    blurFocusedDescendant(root, { activeElement: btn });

    expect(blurred).toBe(true);
    root.remove();
  });

  test('no-ops when activeElement is outside root', () => {
    const root = document.createElement('div');
    const outside = document.createElement('button');
    let blurred = false;
    outside.blur = () => {
      blurred = true;
    };
    document.body.appendChild(root);
    document.body.appendChild(outside);

    blurFocusedDescendant(root, { activeElement: outside });

    expect(blurred).toBe(false);
    root.remove();
    outside.remove();
  });

  test('no-ops for null root or non-blurable activeElement', () => {
    blurFocusedDescendant(null);
    blurFocusedDescendant(document.createElement('div'), { activeElement: null });
    blurFocusedDescendant(document.createElement('div'), { activeElement: {} as any });
  });
});
