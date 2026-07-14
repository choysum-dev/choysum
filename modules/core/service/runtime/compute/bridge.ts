// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getJsCtxRoot } from '../context/source';
import { asObjectRecord } from '../../../utils/object';
import type { ObjectRecord } from '../../../utils/types';

export type BridgeKind = 'sql' | 'search' | 'inverse';

export interface BridgeFrame<TPayload = unknown> {
  id: number;
  kind: BridgeKind;
  instance: object;
  payload: TPayload;
  expired: boolean;
}

const STACK_KEY = Symbol.for('choysum.runtime.compute.bridge.stack');
const LAST_FRAME_KEY = Symbol.for('choysum.runtime.compute.bridge.lastFrameByInstance');

type BridgeCarrier = ObjectRecord & {
  [STACK_KEY]?: BridgeFrame[];
  [LAST_FRAME_KEY]?: WeakMap<object, BridgeFrame>;
};

const processCarrier: BridgeCarrier = {};
let bridgeFrameSeq = 0;

function getBridgeCarrier(): BridgeCarrier {
  return (asObjectRecord(getJsCtxRoot()) as BridgeCarrier | undefined) || processCarrier;
}

function getBridgeStack(carrier: BridgeCarrier): BridgeFrame[] {
  if (!Array.isArray(carrier[STACK_KEY])) {
    carrier[STACK_KEY] = [];
  }
  return carrier[STACK_KEY] as BridgeFrame[];
}

function getLastFrameMap(carrier: BridgeCarrier): WeakMap<object, BridgeFrame> {
  if (!(carrier[LAST_FRAME_KEY] instanceof WeakMap)) {
    carrier[LAST_FRAME_KEY] = new WeakMap<object, BridgeFrame>();
  }
  return carrier[LAST_FRAME_KEY] as WeakMap<object, BridgeFrame>;
}

function isPromiseLike<T = unknown>(value: unknown): value is Promise<T> {
  return !!value && typeof (value as { then?: unknown }).then === 'function';
}

function normalizeKind(kind: unknown): BridgeKind {
  if (kind === 'sql' || kind === 'search' || kind === 'inverse') return kind;
  throw new Error(`BRIDGE_CONTEXT_UNAVAILABLE: unknown bridge kind "${String(kind || '')}"`);
}

function assertInstance(instance: unknown): asserts instance is object {
  if (!instance || (typeof instance !== 'object' && typeof instance !== 'function')) {
    throw new Error('BRIDGE_CONTEXT_UNAVAILABLE: bridge instance is not available');
  }
}

function formatInstanceLabel(instance: object): string {
  const record = asObjectRecord(instance);
  if (!record) return 'Unknown';

  const ctor = record.constructor as { name?: unknown } | undefined;
  const ctorName = typeof ctor?.name === 'string' ? ctor.name.trim() : '';
  return ctorName || 'Unknown';
}

export function enterBridgeFrame<TPayload>(instance: object, kind: BridgeKind, payload: TPayload): BridgeFrame<TPayload> {
  assertInstance(instance);
  const normalizedKind = normalizeKind(kind);
  const carrier = getBridgeCarrier();
  const stack = getBridgeStack(carrier);
  const frame: BridgeFrame<TPayload> = {
    id: ++bridgeFrameSeq,
    kind: normalizedKind,
    instance,
    payload,
    expired: false,
  };

  stack.push(frame as BridgeFrame);
  getLastFrameMap(carrier).set(instance, frame as BridgeFrame);
  return frame;
}

export function exitBridgeFrame(frame: BridgeFrame | undefined): void {
  if (!frame || frame.expired) return;

  const carrier = getBridgeCarrier();
  const stack = getBridgeStack(carrier);

  for (let index = stack.length - 1; index >= 0; index--) {
    if (stack[index] === frame) {
      stack.splice(index, 1);
      break;
    }
  }

  frame.expired = true;
  getLastFrameMap(carrier).set(frame.instance, frame);
}

export function currentBridgeFrame<TPayload>(instance: object, kind: BridgeKind): TPayload {
  assertInstance(instance);
  const normalizedKind = normalizeKind(kind);
  const carrier = getBridgeCarrier();
  const stack = getBridgeStack(carrier);
  let mismatchedKind: BridgeKind | undefined;

  for (let index = stack.length - 1; index >= 0; index--) {
    const frame = stack[index];
    if (frame.instance !== instance) continue;

    if (frame.kind !== normalizedKind) {
      if (!mismatchedKind) {
        mismatchedKind = frame.kind;
      }
      continue;
    }

    if (frame.expired) {
      throw new Error(`BRIDGE_CONTEXT_EXPIRED: ${normalizedKind} bridge context has expired`);
    }

    return frame.payload as TPayload;
  }

  if (mismatchedKind) {
    throw new Error(
      `BRIDGE_CONTEXT_KIND_MISMATCH: requested ${normalizedKind} bridge but active context is ${mismatchedKind} for ${formatInstanceLabel(instance)}`
    );
  }

  const lastFrame = getLastFrameMap(carrier).get(instance);
  if (lastFrame?.expired && lastFrame.kind === normalizedKind) {
    throw new Error(`BRIDGE_CONTEXT_EXPIRED: ${normalizedKind} bridge context has expired`);
  }

  if (stack.length > 0) {
    throw new Error(`BRIDGE_CONTEXT_INSTANCE_MISMATCH: requested ${normalizedKind} bridge for ${formatInstanceLabel(instance)} outside of its active frame`);
  }

  throw new Error(`BRIDGE_CONTEXT_UNAVAILABLE: ${normalizedKind} bridge context is unavailable`);
}

export function withBridgeFrame<TPayload, TResult>(instance: object, kind: BridgeKind, payload: TPayload, run: () => TResult): TResult {
  const frame = enterBridgeFrame(instance, kind, payload);

  try {
    const result = run();
    if (isPromiseLike(result)) {
      return result.finally(() => {
        exitBridgeFrame(frame);
      }) as TResult;
    }

    exitBridgeFrame(frame);
    return result;
  } catch (error) {
    exitBridgeFrame(frame);
    throw error;
  }
}
