// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { track } from './tracking';

describe('tracking', () => {
  it('tracks nested object field mutations without replacing the whole object', () => {
    const draft = track({
      Preferences: {
        theme: 'light',
        notifications: { email: true },
      },
    });

    draft.Preferences.theme = 'dark';
    draft.Preferences.notifications.email = false;

    expect(draft.hasChanges()).toBe(true);
    expect(draft.getChanges()).toEqual({
      Preferences: {
        theme: 'dark',
        notifications: { email: false },
      },
    });

    draft.resetChanges();

    expect(draft.hasChanges()).toBe(false);
    expect(draft.Preferences.theme).toBe('light');
    expect(draft.Preferences.notifications.email).toBe(true);
  });

  it('tracks relation item updates without mutating the original baseline', () => {
    const draft = track<{ Lines: Array<{ Id?: string; Name: string }> }>({
      Lines: [{ Id: 'line-1', Name: 'Old Name' }],
    });

    draft.Lines[0].Name = 'New Name';

    expect(draft.getChanges()).toEqual({
      Lines: {
        update: [{ Id: 'line-1', Name: 'New Name' }],
      },
    });

    draft.resetChanges();

    expect(draft.hasChanges()).toBe(false);
    expect(draft.Lines[0].Name).toBe('Old Name');
  });

  it('clears relation dirty state when array is restored to its original shape', () => {
    const draft = track<{ Lines: Array<{ Id?: string; Name: string }> }>({
      Lines: [{ Id: 'line-1', Name: 'Persisted' }],
    });

    draft.Lines.push({ Name: 'Draft Row' });
    expect(draft.getChanges()).toEqual({
      Lines: {
        create: [{ Name: 'Draft Row' }],
      },
    });

    draft.Lines.pop();

    expect(draft.hasChanges()).toBe(false);
    expect(draft.getChanges()).toEqual({});
  });

  it('popChanges returns pending updates and resets tracking state', () => {
    const draft = track({ Name: 'A' });

    draft.Name = 'B';
    const popped = draft.popChanges();

    expect(popped).toEqual({ Name: 'B' });
    expect(draft.hasChanges()).toBe(false);
    expect(draft.getChanges()).toEqual({});
    expect(draft.Name).toBe('A');
  });

  it('tracks relation create delete and update operations in one payload', () => {
    const draft = track<{ Lines: Array<{ Id?: string; Name: string }> }>({
      Lines: [
        { Id: 'line-1', Name: 'Old 1' },
        { Id: 'line-2', Name: 'Old 2' },
      ],
    });

    draft.Lines[0].Name = 'New 1';
    draft.Lines.splice(1, 1);
    draft.Lines.push({ Name: 'Draft' });

    expect(draft.getChanges()).toEqual({
      Lines: {
        create: [{ Name: 'Draft' }],
        delete: [{ Id: 'line-2' }],
        update: [{ Id: 'line-1', Name: 'New 1' }],
      },
    });
  });

  it('tracks direct array index assignment and deletion via array proxy traps', () => {
    const draft = track<{ Lines: Array<{ Id?: string; Name: string }> }>({
      Lines: [{ Id: 'line-1', Name: 'Old 1' }],
    });

    draft.Lines[0] = { Id: 'line-1', Name: 'Changed' };
    expect(draft.getChanges()).toEqual({
      Lines: {
        update: [{ Id: 'line-1', Name: 'Changed' }],
      },
    });

    delete (draft.Lines as any)[0];
    expect(draft.getChanges()).toEqual({
      Lines: {
        delete: [{ Id: 'line-1' }],
      },
    });
  });

  it('tracks date field updates by value and ignores private proxy fields', () => {
    const draft = track({
      UpdatedAt: new Date('2024-01-01T00:00:00.000Z'),
      Name: 'n1',
    });

    draft.UpdatedAt = new Date('2024-01-02T00:00:00.000Z');
    (draft as any)._internalNote = 'not-a-model-field';

    expect(draft.getChanges()).toEqual({
      UpdatedAt: new Date('2024-01-02T00:00:00.000Z'),
    });
  });

  it('tracks deleteProperty on nested object proxy and can restore to clean state', () => {
    const draft = track({
      Meta: { level: 1, nested: { enabled: true } },
    });

    delete (draft.Meta.nested as any).enabled;
    expect(draft.getChanges()).toEqual({
      Meta: { level: 1, nested: {} },
    });

    draft.Meta.nested.enabled = true;
    expect(draft.hasChanges()).toBe(false);
  });
});
