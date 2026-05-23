// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { asObjectRecord } from '../../../utils/object';
import type { ObjectRecord } from '../../../utils/types';

/* Shared lightweight timing helper. */
export class Perf {
  static now(): number {
    const globalRecord = asObjectRecord(globalThis);
    const performanceRecord = asObjectRecord(globalRecord?.performance);
    const nowFn = performanceRecord?.now;
    if (typeof nowFn === 'function') {
      const value = nowFn.call(performanceRecord);
      if (typeof value === 'number') {
        return value;
      }
    }
    return Date.now();
  }

  static base(): { id?: string; service?: string } {
    try {
      const globalRecord = asObjectRecord(globalThis);
      const choysumRecord = asObjectRecord(globalRecord?.$choysum);
      const reqRecord = asObjectRecord(choysumRecord?.request);

      const id = reqRecord?.id;
      const service = reqRecord?.service;
      return {
        id: id === undefined || id === null ? undefined : String(id),
        service: service === undefined || service === null ? undefined : String(service),
      };
    } catch {
      return {};
    }
  }

  static label(stage: string): string {
    const { id, service } = this.base();
    return `[perf] ${service ?? '-'}#${id ?? '-'} :: ${stage}`;
  }

  static start(stage: string): { stage: string; t0: number; label: string } {
    const t0 = this.now();
    const label = this.label(stage);
    // Optional: log the start point.
    // console.debug(`${label} - start`);
    return { stage, t0, label };
  }

  static end(token: { stage: string; t0: number; label: string } | undefined, extra?: ObjectRecord): number {
    if (!token) return 0;
    const t1 = this.now();
    const dur = t1 - token.t0;
    try {
      const ext = extra ? ` ${JSON.stringify(extra)}` : '';
      console.debug(`${token.label} - ${dur.toFixed(3)}ms${ext}`);
    } catch {
      console.debug(`${token.label} - ${dur.toFixed(3)}ms`);
    }
    return dur;
  }

  static async wrap<T>(stage: string, fn: () => Promise<T> | T, extra?: (res: T) => ObjectRecord | undefined): Promise<T> {
    const t = this.start(stage);
    try {
      const res = await fn();
      this.end(t, extra ? extra(res) : undefined);
      return res;
    } catch (e) {
      const message = e instanceof Error ? e.message : String(e);
      this.end(t, { error: message });
      throw e;
    }
  }
}
