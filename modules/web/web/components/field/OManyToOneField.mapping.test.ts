// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function source(): string {
  return readFileSync(resolve(__dirname, 'OManyToOneField.vue'), 'utf8');
}

describe('OManyToOneField mapping/contract', () => {
  it('exports value click payload and emit contract', () => {
    const s = source();

    expect(s).toContain('export type ValueClickPayload<T = any> = {');
    expect(s).toContain("(e: 'value-click', payload: ValueClickPayload<any>): void;");
    expect(s).toContain("source: 'display';");
  });

  it('supports auto value click mode by listener detection', () => {
    const s = source();

    expect(s).toContain("valueClickable?: boolean | 'auto';");
    expect(s).toContain("valueClickable: 'auto',");
    expect(s).toContain("return Boolean(p.onValueClick || p['onValue-click']);");
    expect(s).toContain('if (props.valueClickable === true) return true;');
    expect(s).toContain('if (props.valueClickable === false) return false;');
  });

  it('wires display mode click and keyboard activation', () => {
    const s = source();

    expect(s).toContain('@click="onDisplayValueClick(fieldValue().value as any, $event)"');
    expect(s).toContain('@keydown="onDisplayValueKeydown(fieldValue().value as any, $event)"');
    expect(s).toContain("if (event.key !== 'Enter' && event.key !== ' ') return;");
    expect(s).toContain(
      "emit('value-click', { id, item: val && typeof val === 'object' ? val : null, label: getDisplayLabel(val), source: 'display', event });"
    );
    expect(s).toContain('.o-field-display-text--clickable');
  });
});
