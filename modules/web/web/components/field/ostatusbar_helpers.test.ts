// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import {
  applyStatusbarSelect,
  canSelectStatusbarValue,
  currentFromFieldValue,
  currentFromRowRef,
  fromStatusbarView,
  gateBeforeChange,
  normalizeSegmentedModelValue,
  pickRootOnchangeSelection,
  resolveStatusbarOptions,
  resolveStatusbarWhitelist,
  toSegmentedOptions,
  toStatusbarView,
  validateStatusbarValue,
} from './ostatusbar_helpers';

const meta = [
  { value: 'draft', label: 'Draft' },
  { value: 'confirmed', label: 'Confirmed' },
  { value: 'done', label: 'Done' },
  { value: 'cancel', label: 'Cancelled' },
];

describe('toStatusbarView / fromStatusbarView / normalizeSegmentedModelValue', () => {
  it('normalizes view and model values', () => {
    expect(toStatusbarView(null)).toBeNull();
    expect(toStatusbarView(undefined)).toBeNull();
    expect(toStatusbarView('done')).toBe('done');
    expect(toStatusbarView(1)).toBe('1');
    expect(fromStatusbarView(null)).toBeNull();
    expect(fromStatusbarView('done')).toBe('done');
    expect(normalizeSegmentedModelValue(null)).toBeUndefined();
    expect(normalizeSegmentedModelValue('')).toBeUndefined();
    expect(normalizeSegmentedModelValue('done')).toBe('done');
  });
});

describe('resolveStatusbarWhitelist', () => {
  it('prefers statusbarVisible over selection', () => {
    expect(resolveStatusbarWhitelist(['a'], ['b'])).toEqual(['a']);
    expect(resolveStatusbarWhitelist([], ['b'])).toEqual(['b']);
    expect(resolveStatusbarWhitelist(undefined, [])).toBeNull();
    expect(resolveStatusbarWhitelist(null, null)).toBeNull();
  });
});

describe('pickRootOnchangeSelection', () => {
  it('returns null for missing / unmatched payloads', () => {
    expect(pickRootOnchangeSelection(null, 'State')).toBeNull();
    expect(pickRootOnchangeSelection({ selection: [] }, 'State')).toBeNull();
    expect(pickRootOnchangeSelection({ selection: [{ field: 'Other', selection: ['a'] }] }, 'State')).toBeNull();
  });

  it('reads selection and optional disabled', () => {
    expect(
      pickRootOnchangeSelection(
        { selection: [{ field: 'State', selection: ['draft', 'done'], disabled: ['done'] }] },
        'State'
      )
    ).toEqual({ values: ['draft', 'done'], disabled: ['done'] });
    expect(pickRootOnchangeSelection({ selection: [{ field: 'State', selection: 'bad' }] }, 'State')).toBeNull();
    expect(pickRootOnchangeSelection({ selection: [{ field: 'State', selection: [] }] }, 'State')).toEqual({
      values: [],
      disabled: undefined,
    });
  });
});

describe('currentFromRowRef / currentFromFieldValue', () => {
  it('reads current from row refs and field values', () => {
    expect(currentFromRowRef(null, 'State')).toBeNull();
    expect(currentFromRowRef({ State: 'done' }, '')).toBeNull();
    expect(currentFromRowRef({ State: 'done' }, 'State')).toBe('done');
    expect(currentFromRowRef(() => ({ value: { State: 'draft' } }), 'State')).toBe('draft');
    expect(currentFromRowRef(() => ({ State: 'confirmed' }), 'State')).toBe('confirmed');
    expect(currentFromRowRef(() => {
      throw new Error('boom');
    }, 'State')).toBeNull();
    expect(currentFromRowRef({ value: null }, 'State')).toBeNull();
    expect(currentFromFieldValue(null)).toBeNull();
    expect(currentFromFieldValue('')).toBeNull();
    expect(currentFromFieldValue('done')).toBe('done');
    const bad = {
      toString() {
        throw new Error('nope');
      },
      valueOf() {
        throw new Error('nope');
      },
    };
    expect(currentFromFieldValue(bad)).toBeNull();
  });
});

describe('resolveStatusbarOptions', () => {
  it('returns meta options in order by default', () => {
    expect(resolveStatusbarOptions({ meta }).map(o => o.value)).toEqual(['draft', 'confirmed', 'done', 'cancel']);
  });

  it('handles non-array meta, blanks, duplicates, and missing labels', () => {
    expect(resolveStatusbarOptions({ meta: null as any })).toEqual([]);
    expect(
      resolveStatusbarOptions({
        meta: [
          { value: '', label: 'x' },
          { value: 'a', label: undefined as any },
          { value: 'a', label: 'dup' },
          null as any,
        ],
      }).map(o => o)
    ).toEqual([{ value: 'a', label: 'a', disabled: false }]);
  });

  it('applies whitelist order and filters (statusbarVisible)', () => {
    expect(
      resolveStatusbarOptions({
        meta,
        whitelist: ['done', 'draft', null as any, ''],
      }).map(o => o.value)
    ).toEqual(['done', 'draft']);
  });

  it('uses bare whitelist when pool is empty', () => {
    expect(resolveStatusbarOptions({ meta: [], whitelist: ['x', 'y'] }).map(o => o.value)).toEqual(['x', 'y']);
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

  it('falls back current label when not in meta', () => {
    expect(
      resolveStatusbarOptions({
        meta: [],
        whitelist: ['a'],
        current: 'orphan',
        onchangeDisabled: ['orphan'],
      })
    ).toEqual([
      { value: 'a', label: 'a', disabled: false },
      { value: 'orphan', label: 'orphan', disabled: true },
    ]);
  });

  it('ignores empty current and already-visible current', () => {
    expect(resolveStatusbarOptions({ meta, current: '' }).map(o => o.value)).toEqual([
      'draft',
      'confirmed',
      'done',
      'cancel',
    ]);
    expect(resolveStatusbarOptions({ meta, current: 'draft' }).map(o => o.value)).toEqual([
      'draft',
      'confirmed',
      'done',
      'cancel',
    ]);
  });

  it('intersects onchange values with meta and marks disabled', () => {
    const opts = resolveStatusbarOptions({
      meta,
      onchangeValues: ['draft', 'done', 'unknown', '', null as any, 'draft'],
      onchangeDisabled: ['done'],
      current: 'draft',
    });
    expect(opts.map(o => ({ value: o.value, disabled: o.disabled }))).toEqual([
      { value: 'draft', disabled: false },
      { value: 'done', disabled: true },
    ]);
  });

  it('honors explicit empty onchange domain (no fallthrough to meta)', () => {
    expect(resolveStatusbarOptions({ meta, onchangeValues: [] }).map(o => o.value)).toEqual([]);
    expect(
      resolveStatusbarOptions({
        meta,
        onchangeValues: [],
        whitelist: ['draft', 'done'],
        current: 'draft',
      }).map(o => o.value)
    ).toEqual(['draft']);
  });

  it('keeps onchange values when meta is empty', () => {
    expect(
      resolveStatusbarOptions({
        meta: [],
        onchangeValues: ['a', 'b'],
      }).map(o => o.value)
    ).toEqual(['a', 'b']);
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

describe('toSegmentedOptions / canSelect / validate', () => {
  it('maps and validates options', () => {
    const opts = resolveStatusbarOptions({ meta, onchangeDisabled: ['done'] });
    expect(toSegmentedOptions(opts)[0]).toEqual({ label: 'Draft', value: 'draft', disabled: false });
    expect(canSelectStatusbarValue('draft', opts)).toBe(true);
    expect(canSelectStatusbarValue('done', opts)).toBe(false);
    expect(canSelectStatusbarValue('nope', opts)).toBe(false);

    const msgs = { mustBeString: 'str', invalid: (v: string) => `bad:${v}` };
    expect(validateStatusbarValue(null, opts, msgs)).toBeNull();
    expect(validateStatusbarValue('', opts, msgs)).toBeNull();
    expect(validateStatusbarValue(1, opts, msgs)?.message).toBe('str');
    expect(validateStatusbarValue('nope', opts, msgs)?.message).toBe('bad:nope');
    expect(validateStatusbarValue('draft', opts, msgs)).toBeNull();
  });
});

describe('gateBeforeChange / applyStatusbarSelect', () => {
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

  it('covers applyStatusbarSelect branches', async () => {
    const opts = resolveStatusbarOptions({ meta, onchangeDisabled: ['done'] });
    const write = vi.fn();
    expect(
      await applyStatusbarSelect({
        interactive: false,
        pending: false,
        nextRaw: 'confirmed',
        current: 'draft',
        options: opts,
        write,
      })
    ).toBe('skipped');
    expect(
      await applyStatusbarSelect({
        interactive: true,
        pending: true,
        nextRaw: 'confirmed',
        current: 'draft',
        options: opts,
        write,
      })
    ).toBe('skipped');
    expect(
      await applyStatusbarSelect({
        interactive: true,
        pending: false,
        nextRaw: null,
        current: 'draft',
        options: opts,
        write,
      })
    ).toBe('skipped');
    expect(
      await applyStatusbarSelect({
        interactive: true,
        pending: false,
        nextRaw: 'draft',
        current: 'draft',
        options: opts,
        write,
      })
    ).toBe('skipped');
    expect(
      await applyStatusbarSelect({
        interactive: true,
        pending: false,
        nextRaw: 'done',
        current: 'draft',
        options: opts,
        write,
      })
    ).toBe('skipped');
    expect(
      await applyStatusbarSelect({
        interactive: true,
        pending: false,
        nextRaw: 'confirmed',
        current: 'draft',
        options: opts,
        beforeChange: () => false,
        write,
      })
    ).toBe('cancelled');
    expect(
      await applyStatusbarSelect({
        interactive: true,
        pending: false,
        nextRaw: 'confirmed',
        current: 'draft',
        options: opts,
        write,
      })
    ).toBe('written');
    expect(write).toHaveBeenCalledWith('confirmed');
  });
});
