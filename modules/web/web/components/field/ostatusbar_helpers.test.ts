// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { gateBeforeChange, resolveStatusbarOptions } from './ostatusbar_helpers';

const meta = [
  { value: 'draft', label: 'Draft' },
  { value: 'confirmed', label: 'Confirmed' },
  { value: 'done', label: 'Done' },
  { value: 'cancel', label: 'Cancelled' },
];

describe('resolveStatusbarOptions', () => {
  it('returns meta options in order by default', () => {
    expect(resolveStatusbarOptions({ meta }).map(o => o.value)).toEqual(['draft', 'confirmed', 'done', 'cancel']);
  });

  it('applies whitelist order and filters (statusbarVisible)', () => {
    expect(
      resolveStatusbarOptions({
        meta,
        whitelist: ['done', 'draft'],
      }).map(o => o.value)
    ).toEqual(['done', 'draft']);
  });

  it('keeps current value when missing from whitelist (D5 fallback)', () => {
    const opts = resolveStatusbarOptions({
      meta,
      whitelist: ['draft', 'done'],
      current: 'cancel',
    });
    expect(opts.map(o => o.value)).toEqual(['draft', 'done', 'cancel']);
    expect(opts[2]?.label).toBe('Cancelled');
  });

  it('intersects onchange values with meta and marks disabled', () => {
    const opts = resolveStatusbarOptions({
      meta,
      onchangeValues: ['draft', 'done', 'unknown'],
      onchangeDisabled: ['done'],
      current: 'draft',
    });
    expect(opts.map(o => ({ value: o.value, disabled: o.disabled }))).toEqual([
      { value: 'draft', disabled: false },
      { value: 'done', disabled: true },
    ]);
  });

  it('applies whitelist after onchange filter', () => {
    expect(
      resolveStatusbarOptions({
        meta,
        onchangeValues: ['draft', 'confirmed', 'done'],
        whitelist: ['done', 'draft'],
      }).map(o => o.value)
    ).toEqual(['done', 'draft']);
  });
});

describe('gateBeforeChange', () => {
  it('allows when hook is omitted', async () => {
    expect(await gateBeforeChange(undefined, 'done', 'draft')).toBe(true);
  });

  it('allows only when hook returns true', async () => {
    expect(await gateBeforeChange(() => true, 'done', 'draft')).toBe(true);
    expect(await gateBeforeChange(() => false, 'done', 'draft')).toBe(false);
    expect(await gateBeforeChange(() => undefined as any, 'done', 'draft')).toBe(false);
  });

  it('cancels on throw / reject', async () => {
    expect(
      await gateBeforeChange(() => {
        throw new Error('nope');
      }, 'done', 'draft')
    ).toBe(false);
    expect(await gateBeforeChange(async () => Promise.reject(new Error('nope')), 'done', 'draft')).toBe(false);
  });

  it('awaits async hooks', async () => {
    const hook = vi.fn(async () => true);
    expect(await gateBeforeChange(hook, 'done', 'draft')).toBe(true);
    expect(hook).toHaveBeenCalledWith('done', 'draft');
  });
});
