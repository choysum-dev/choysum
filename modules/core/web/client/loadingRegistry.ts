// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Client loading registry (chrome bar + userBusy / slow-hint + surface/control).
 * Product rules: modules/web/web/docs/choy-loading-feedback.md
 */
import { computed, ref, toValue, type ComputedRef, type MaybeRefOrGetter, type Ref } from 'vue';

/** View/page content busy presentation. */
export type SurfaceMode = 'stale' | 'skeleton' | 'block';

/** Intent layer for a user-visible or silent operation. */
export type LoadingLayer = 'surface' | 'control' | 'chrome';

export type LoadingOpts = {
  layer?: LoadingLayer;
  target?: string;
  mode?: SurfaceMode;
  /** Skip bar and slow-hint; used for NameSearch / option loads. */
  silent?: boolean;
  /** Optional label for control busy (e.g. Saving…). */
  pendingLabel?: string;
};

type StackFrame = {
  silent: boolean;
  layer: LoadingLayer | 'silent';
  target?: string;
  mode?: SurfaceMode;
  pendingLabel?: string;
  /** True when this frame already owns bar / user-busy counts. */
  ownsBar: boolean;
  ownsUserBusy: boolean;
};

type TargetSurface = { count: number; mode: SurfaceMode };
type TargetControl = { count: number; pendingLabel?: string };

const BAR_DELAY_MS = 200;
/** Delay before first showing the shell slow hint. */
const SLOW_HINT_MS = 3000;
/** Once shown, keep the hint at least this long so it does not flash off with the request. */
const SLOW_HINT_MIN_SHOW_MS = 1500;
/** Drop orphaned navigation bar tickets if no surface settles. */
const NAV_TIMEOUT_MS = 15_000;
/**
 * After route afterEach: if no surface started, drop the nav bar ticket after a
 * short grace. Guest pages (Setup/Login) never open surface — without this the
 * bar (and formerly slow-hint) stuck until NAV_TIMEOUT_MS. Grace must cover
 * Guest→App shell mount + List/Search first-frame surface under slow nets so
 * the top bar does not drop before the first authenticated surface owns it.
 */
const NAV_IDLE_SETTLE_MS = 2000;

const stack: StackFrame[] = [];

let barCount = 0;
let userBusyCount = 0;
let navTicketActive = false;
let navTimeoutId: ReturnType<typeof setTimeout> | null = null;
let navIdleSettleId: ReturnType<typeof setTimeout> | null = null;

const surfaceByTarget = new Map<string, TargetSurface>();
const controlByTarget = new Map<string, TargetControl>();

/** RPC span → whether it opened a default chrome ticket (no outer frame). */
const rpcChromeTickets = new Map<string, { bar: boolean; userBusy: boolean }>();

const barPending = ref(false);
const userBusyPending = ref(false);
const showBar = ref(false);
const showSlowHint = ref(false);
const surfaceEpoch = ref(0);
const controlEpoch = ref(0);
/** Bumps when nav ticket starts/settles so handoff overlays can react. */
const navEpoch = ref(0);

let barDelayTimer: ReturnType<typeof setTimeout> | null = null;
let slowHintTimer: ReturnType<typeof setTimeout> | null = null;
let slowHintHideTimer: ReturnType<typeof setTimeout> | null = null;
/** Timestamp when the slow hint became visible (for min-show dwell). */
let slowHintShownAt = 0;

function bumpSurface(): void {
  surfaceEpoch.value += 1;
}

function bumpControl(): void {
  controlEpoch.value += 1;
}

function bumpNav(): void {
  navEpoch.value += 1;
}

function clearBarDelay(): void {
  if (barDelayTimer != null) {
    clearTimeout(barDelayTimer);
    barDelayTimer = null;
  }
}

function clearSlowHintDelay(): void {
  if (slowHintTimer != null) {
    clearTimeout(slowHintTimer);
    slowHintTimer = null;
  }
}

function clearSlowHintHideDelay(): void {
  if (slowHintHideTimer != null) {
    clearTimeout(slowHintHideTimer);
    slowHintHideTimer = null;
  }
}

function hideSlowHintNow(): void {
  clearSlowHintHideDelay();
  showSlowHint.value = false;
  slowHintShownAt = 0;
}

function revealSlowHint(): void {
  clearSlowHintHideDelay();
  showSlowHint.value = true;
  slowHintShownAt = Date.now();
}

function syncBarDisplay(): void {
  const pending = barCount > 0;
  barPending.value = pending;
  if (!pending) {
    clearBarDelay();
    showBar.value = false;
    return;
  }
  if (showBar.value) return;
  if (barDelayTimer != null) return;
  barDelayTimer = setTimeout(() => {
    barDelayTimer = null;
    if (barCount > 0) showBar.value = true;
  }, BAR_DELAY_MS);
}

function syncSlowHintDisplay(): void {
  const pending = userBusyCount > 0;
  userBusyPending.value = pending;
  if (pending) {
    // Busy resumed: cancel a pending hide and keep / schedule show.
    clearSlowHintHideDelay();
    if (showSlowHint.value) return;
    if (slowHintTimer != null) return;
    slowHintTimer = setTimeout(() => {
      slowHintTimer = null;
      if (userBusyCount > 0) revealSlowHint();
    }, SLOW_HINT_MS);
    return;
  }

  // Idle: cancel a pending first-show; if already visible, dwell for min-show.
  clearSlowHintDelay();
  if (!showSlowHint.value) {
    hideSlowHintNow();
    return;
  }
  if (slowHintHideTimer != null) return;
  const shownFor = slowHintShownAt > 0 ? Date.now() - slowHintShownAt : SLOW_HINT_MIN_SHOW_MS;
  const remain = Math.max(0, SLOW_HINT_MIN_SHOW_MS - shownFor);
  if (remain === 0) {
    hideSlowHintNow();
    return;
  }
  slowHintHideTimer = setTimeout(() => {
    slowHintHideTimer = null;
    if (userBusyCount === 0) hideSlowHintNow();
  }, remain);
}

function addBar(n = 1): void {
  barCount += n;
  syncBarDisplay();
}

function removeBar(n = 1): void {
  barCount = Math.max(0, barCount - n);
  syncBarDisplay();
}

function addUserBusy(n = 1): void {
  userBusyCount += n;
  syncSlowHintDisplay();
}

function removeUserBusy(n = 1): void {
  userBusyCount = Math.max(0, userBusyCount - n);
  syncSlowHintDisplay();
}

function acquireSurface(target: string, mode: SurfaceMode): void {
  const cur = surfaceByTarget.get(target);
  if (cur) {
    cur.count += 1;
    cur.mode = mode;
  } else {
    surfaceByTarget.set(target, { count: 1, mode });
  }
  bumpSurface();
}

function releaseSurface(target: string): void {
  const cur = surfaceByTarget.get(target);
  if (!cur) return;
  cur.count -= 1;
  if (cur.count <= 0) surfaceByTarget.delete(target);
  bumpSurface();
}

function acquireControl(target: string, pendingLabel?: string): void {
  const cur = controlByTarget.get(target);
  if (cur) {
    cur.count += 1;
    if (pendingLabel !== undefined) cur.pendingLabel = pendingLabel;
  } else {
    controlByTarget.set(target, { count: 1, pendingLabel });
  }
  bumpControl();
}

function releaseControl(target: string): void {
  const cur = controlByTarget.get(target);
  if (!cur) return;
  cur.count -= 1;
  if (cur.count <= 0) controlByTarget.delete(target);
  bumpControl();
}

function clearNavTimeout(): void {
  if (navTimeoutId != null) {
    clearTimeout(navTimeoutId);
    navTimeoutId = null;
  }
}

function clearNavIdleSettle(): void {
  if (navIdleSettleId != null) {
    clearTimeout(navIdleSettleId);
    navIdleSettleId = null;
  }
}

function activeSurfaceCount(): number {
  let n = 0;
  for (const cur of surfaceByTarget.values()) {
    if (cur.count > 0) n += cur.count;
  }
  return n;
}

/** True while any surface runWithLoading frame holds a target. */
export function hasActiveSurfaceBusy(): boolean {
  return activeSurfaceCount() > 0;
}

/**
 * Effective stack intent: innermost non-default frame wins for silent/layer.
 * Outer surface/control still "owns" nested RPCs so they do not open chrome tickets.
 */
function stackOwnsRpc(): { silent: boolean; underControl: boolean; underSurface: boolean; underChrome: boolean } {
  let silent = false;
  let underControl = false;
  let underSurface = false;
  let underChrome = false;
  for (const frame of stack) {
    if (frame.silent || frame.layer === 'silent') silent = true;
    if (frame.layer === 'control') underControl = true;
    if (frame.layer === 'surface') underSurface = true;
    if (frame.layer === 'chrome') underChrome = true;
  }
  return { silent, underControl, underSurface, underChrome };
}

/**
 * Run `fn` under a loading intent. Outer frames cover nested RPCs (no double bar).
 * Does not write W3C baggage — client-only stack.
 */
export async function runWithLoading<T>(opts: LoadingOpts, fn: () => Promise<T>): Promise<T> {
  const silent = !!opts.silent;
  const layer: LoadingLayer | 'silent' = silent ? 'silent' : opts.layer ?? 'chrome';
  const target = opts.target?.trim() || undefined;
  const mode = opts.mode ?? 'stale';
  const pendingLabel = opts.pendingLabel;

  const ownsBar = !silent && (layer === 'surface' || layer === 'chrome');
  const ownsUserBusy = !silent && (layer === 'surface' || layer === 'control' || layer === 'chrome');

  const frame: StackFrame = {
    silent,
    layer,
    target,
    mode,
    pendingLabel,
    ownsBar,
    ownsUserBusy,
  };
  stack.push(frame);

  if (ownsBar) addBar();
  if (ownsUserBusy) addUserBusy();
  if (!silent && layer === 'surface' && target) acquireSurface(target, mode);
  if (!silent && layer === 'control' && target) acquireControl(target, pendingLabel);

  try {
    return await fn();
  } catch (e) {
    // Explicit catch so QuickJS marks the rejection handled before finally cleanup.
    throw e;
  } finally {
    stack.pop();
    if (ownsBar) removeBar();
    if (ownsUserBusy) removeUserBusy();
    if (!silent && layer === 'surface' && target) {
      releaseSurface(target);
      // First surface after navigation may settle the nav bar ticket.
      settleNavigationIfActive();
    }
    if (!silent && layer === 'control' && target) releaseControl(target);
  }
}

/** Synchronous variant for rare sync work under a loading frame. */
export function runWithLoadingSync<T>(opts: LoadingOpts, fn: () => T): T {
  const silent = !!opts.silent;
  const layer: LoadingLayer | 'silent' = silent ? 'silent' : opts.layer ?? 'chrome';
  const target = opts.target?.trim() || undefined;
  const mode = opts.mode ?? 'stale';
  const pendingLabel = opts.pendingLabel;
  const ownsBar = !silent && (layer === 'surface' || layer === 'chrome');
  const ownsUserBusy = !silent && (layer === 'surface' || layer === 'control' || layer === 'chrome');

  stack.push({
    silent,
    layer,
    target,
    mode,
    pendingLabel,
    ownsBar,
    ownsUserBusy,
  });
  if (ownsBar) addBar();
  if (ownsUserBusy) addUserBusy();
  if (!silent && layer === 'surface' && target) acquireSurface(target, mode);
  if (!silent && layer === 'control' && target) acquireControl(target, pendingLabel);
  try {
    return fn();
  } finally {
    stack.pop();
    if (ownsBar) removeBar();
    if (ownsUserBusy) removeUserBusy();
    if (!silent && layer === 'surface' && target) {
      releaseSurface(target);
      settleNavigationIfActive();
    }
    if (!silent && layer === 'control' && target) releaseControl(target);
  }
}

/**
 * Router beforeEach: bar + user-busy (slow-hint uses the same time threshold as
 * surface/control — fast nav stays quiet; slow chunk waits can show chef).
 * Kept until first surface ends, idle settle after afterEach, or NAV_TIMEOUT_MS.
 */
export function beginNavigationLoading(): void {
  if (navTicketActive) return;
  navTicketActive = true;
  bumpNav();
  addBar();
  addUserBusy();
  clearNavIdleSettle();
  clearNavTimeout();
  navTimeoutId = setTimeout(() => {
    navTimeoutId = null;
    settleNavigationIfActive();
  }, NAV_TIMEOUT_MS);
}

/**
 * Router afterEach: if the page never opens a surface (Setup/Login/static),
 * drop the nav bar after NAV_IDLE_SETTLE_MS so chrome does not linger on an
 * idle form. Grace covers Guest→App mount + first surface under slow nets;
 * when a surface is already (or soon) busy, leave the ticket for surface finally.
 */
export function scheduleNavigationSettleIfIdle(delayMs: number = NAV_IDLE_SETTLE_MS): void {
  if (!navTicketActive) return;
  clearNavIdleSettle();
  navIdleSettleId = setTimeout(() => {
    navIdleSettleId = null;
    if (!navTicketActive) return;
    if (hasActiveSurfaceBusy()) return;
    settleNavigationIfActive();
  }, delayMs);
}

/** Call when a surface load finishes after navigation (also auto from runWithLoading surface). */
export function settleNavigationIfActive(): void {
  if (!navTicketActive) return;
  navTicketActive = false;
  bumpNav();
  clearNavTimeout();
  clearNavIdleSettle();
  removeBar();
  removeUserBusy();
}

/**
 * Lifecycle hook: unmarked RPCs open a chrome ticket; RPCs under surface/control/silent
 * do not (outer frame already owns feedback).
 */
export function loadingRegistryOnRpcStart(spanId: string): void {
  // Idempotent per span: duplicate start must not double-count bar/userBusy.
  if (rpcChromeTickets.has(spanId)) return;
  const { silent, underControl, underSurface, underChrome } = stackOwnsRpc();
  if (silent || underControl || underSurface || underChrome) {
    rpcChromeTickets.set(spanId, { bar: false, userBusy: false });
    return;
  }
  addBar();
  addUserBusy();
  rpcChromeTickets.set(spanId, { bar: true, userBusy: true });
}

export function loadingRegistryOnRpcFinish(spanId: string): void {
  const ticket = rpcChromeTickets.get(spanId);
  rpcChromeTickets.delete(spanId);
  if (!ticket) return;
  if (ticket.bar) removeBar();
  if (ticket.userBusy) removeUserBusy();
}

export type SurfaceBusySnapshot = { busy: boolean; mode: SurfaceMode | null };
export type ControlBusySnapshot = { busy: boolean; pendingLabel?: string };

export function getSurfaceBusy(target: string): SurfaceBusySnapshot {
  const cur = surfaceByTarget.get(target);
  if (!cur || cur.count <= 0) return { busy: false, mode: null };
  return { busy: true, mode: cur.mode };
}

export function getControlBusy(target: string): ControlBusySnapshot {
  const cur = controlByTarget.get(target);
  if (!cur || cur.count <= 0) return { busy: false };
  return { busy: true, pendingLabel: cur.pendingLabel };
}

/** Reactive: true after bar has been pending ≥ BAR_DELAY_MS. */
export function useLoadingBarVisible(): Ref<boolean> {
  return showBar;
}

/** Reactive: true after user-initiated busy ≥ SLOW_HINT_MS. */
export function useLoadingSlowHintVisible(): Ref<boolean> {
  return showSlowHint;
}

/**
 * True while a navigation ticket is open and no surface has taken over yet.
 * Hosts a full-viewport block so Guest→App (or any route swap) does not look idle/clickable
 * after the leaving page clears its own mask.
 */
export function useNavigationHandoffBusy(): ComputedRef<boolean> {
  return computed(() => {
    void navEpoch.value;
    void surfaceEpoch.value;
    return navTicketActive && !hasActiveSurfaceBusy();
  });
}

/** Reactive: any surface runWithLoading target is busy. */
export function useHasActiveSurfaceBusy(): ComputedRef<boolean> {
  return computed(() => {
    void surfaceEpoch.value;
    return hasActiveSurfaceBusy();
  });
}

export function useSurfaceBusy(target: MaybeRefOrGetter<string>): ComputedRef<SurfaceBusySnapshot> {
  return computed(() => {
    void surfaceEpoch.value;
    return getSurfaceBusy(toValue(target));
  });
}

export function useControlBusy(target: MaybeRefOrGetter<string>): ComputedRef<ControlBusySnapshot> {
  return computed(() => {
    void controlEpoch.value;
    return getControlBusy(toValue(target));
  });
}

/** Test helper: apply showBar / showSlowHint from counters without waiting delays. */
export function flushLoadingDisplayForTests(): void {
  clearBarDelay();
  clearSlowHintDelay();
  clearSlowHintHideDelay();
  showBar.value = barCount > 0;
  if (userBusyCount > 0) {
    revealSlowHint();
  } else {
    hideSlowHintNow();
  }
  barPending.value = barCount > 0;
  userBusyPending.value = userBusyCount > 0;
}

/** Test / teardown helper. */
export function resetLoadingRegistryForTests(): void {
  stack.length = 0;
  barCount = 0;
  userBusyCount = 0;
  navTicketActive = false;
  clearNavTimeout();
  clearNavIdleSettle();
  clearBarDelay();
  clearSlowHintDelay();
  clearSlowHintHideDelay();
  surfaceByTarget.clear();
  controlByTarget.clear();
  rpcChromeTickets.clear();
  barPending.value = false;
  userBusyPending.value = false;
  showBar.value = false;
  showSlowHint.value = false;
  slowHintShownAt = 0;
  surfaceEpoch.value = 0;
  controlEpoch.value = 0;
  navEpoch.value = 0;
}

export const loadingTiming = {
  BAR_DELAY_MS,
  SLOW_HINT_MS,
  SLOW_HINT_MIN_SHOW_MS,
  NAV_TIMEOUT_MS,
  NAV_IDLE_SETTLE_MS,
} as const;

/** Debug / tests: current counters. */
export function getLoadingRegistryDebugState(): {
  barCount: number;
  userBusyCount: number;
  navTicketActive: boolean;
  stackDepth: number;
} {
  return {
    barCount,
    userBusyCount,
    navTicketActive,
    stackDepth: stack.length,
  };
}
