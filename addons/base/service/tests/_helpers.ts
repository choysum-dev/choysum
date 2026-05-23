// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';

export function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const token = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${token}`;
}

function shortToken(size: number): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const base = typeof xid === 'string' && xid.trim() ? xid.trim() : `${Date.now()}${Math.random()}`;
  const raw = String(base)
    .replace(/[^A-Za-z0-9]/g, '')
    .toUpperCase();
  const tail = raw.slice(-size);
  return `${tail}${'X'.repeat(size)}`.slice(0, size);
}

export function currencyCode3(): string {
  const code = shortToken(3);
  return code === 'CNY' ? 'CNX' : code;
}

export function companyCode8(): string {
  return shortToken(8);
}

export function countryCode8(): string {
  return shortToken(8);
}

export function countryCode2(): string {
  return shortToken(2);
}

export async function expectBaseInvalidArgument(fn: () => Promise<any>): Promise<void> {
  try {
    await fn();
    throw new Error('expected InvalidArgument error');
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    const oe = err as ChoysumError;
    expect(oe.domain).toBe('base');
    expect(oe.code).toBe('InvalidArgument');
  }
}

export async function expectBaseNotFound(fn: () => Promise<any>): Promise<void> {
  try {
    await fn();
    throw new Error('expected NotFound error');
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    const oe = err as ChoysumError;
    expect(oe.domain).toBe('base');
    expect(oe.code).toBe('NotFound');
  }
}

export function isOptimisticLockConflict(err: unknown): boolean {
  const message = String((err as Error)?.message || err || '');
  return message.includes('has been modified') || message.includes('modified by another user');
}

export async function updateInstanceWithRetry(
  inst: any,
  pendingChanges: Record<string, unknown>,
  options?: Record<string, unknown>,
  maxAttempts = 5
): Promise<void> {
  let lastErr: unknown;
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    Object.keys(pendingChanges || {}).forEach(key => {
      inst[key] = pendingChanges[key];
    });
    try {
      if (options) {
        await inst.update(options);
      } else {
        await inst.update();
      }
      return;
    } catch (err) {
      lastErr = err;
      if (!isOptimisticLockConflict(err)) {
        throw err;
      }
      if (attempt === maxAttempts) {
        break;
      }
      if (options) {
        await inst.reload(options);
      } else {
        await inst.reload();
      }
    }
  }

  // Last-resort deflake path: if optimistic lock keeps failing, write once
  // via static UpdateById and refresh instance state.
  if (isOptimisticLockConflict(lastErr)) {
    const ctor = inst?.constructor as { UpdateById?: (id: string, values: unknown, fields?: unknown, opts?: unknown) => Promise<unknown> } | undefined;
    const id = String(inst?.Id || '').trim();
    if (ctor && typeof ctor.UpdateById === 'function' && id) {
      await ctor.UpdateById(id, pendingChanges as unknown, ['Id'] as unknown, options as unknown);
      if (typeof inst?.reload === 'function') {
        if (options) {
          await inst.reload(options);
        } else {
          await inst.reload();
        }
      }
      return;
    }
  }

  throw lastErr;
}
