// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, onBeforeUnmount, onMounted, onActivated, ref, type Ref } from 'vue';
import {
  collectAdaptiveHeightObserveTargets,
  computeContainerAdaptiveHeight,
  computeViewportAdaptiveHeight,
} from './adaptiveHeightMath';

export type HeightMode = 'auto' | 'viewport' | 'container' | 'none';

export interface AdaptiveHeightOptions {
  mode?: HeightMode; // default: 'auto'
  containerSelector?: string; // Preferred container selector.
  viewportGap?: number; // Reserved gap from the bottom of the viewport in viewport mode.
  minViewportHeight?: number; // Minimum height used in viewport mode.
  minContainerHeight?: number; // Minimum height used in container mode.
  containerPadding?: number; // Extra padding reserved inside the container.
}

/**
 * useAdaptiveHeight
 * Computes content height from either the viewport or a container with requestAnimationFrame throttling.
 * - auto: use container mode inside .el-dialog__body, otherwise viewport mode
 * - container: compute height from the resolved container and subtract the dialog footer when present
 * - viewport: fill to AppShell scroll canvas (above footer) when nested there;
 *   otherwise distance to the viewport bottom minus viewportGap
 * - none: leave height untouched
 */
export function useAdaptiveHeight(wrapRef: Ref<Element | null>, opts: AdaptiveHeightOptions = {}) {
  const options: Required<Omit<AdaptiveHeightOptions, 'mode' | 'containerSelector'>> & Pick<AdaptiveHeightOptions, 'mode' | 'containerSelector'> = {
    mode: opts.mode ?? 'auto',
    containerSelector: opts.containerSelector,
    viewportGap: opts.viewportGap ?? 12,
    minViewportHeight: opts.minViewportHeight ?? 240,
    minContainerHeight: opts.minContainerHeight ?? 160,
    containerPadding: opts.containerPadding ?? 8,
  } as any;

  const height = ref(0);
  const pxHeight = computed(() => (height.value > 0 ? `${height.value}px` : ''));

  let containerEl: HTMLElement | null = null;
  let observedEls: HTMLElement[] = [];
  let ro: ResizeObserver | null = null;
  let rafId: number | null = null;

  function pickMode(): Exclude<HeightMode, 'none'> {
    const wrap = wrapRef.value;
    const requested = options.mode ?? 'auto';
    if (requested === 'none') return 'viewport'; // Placeholder; callers skip computation later.
    if (requested !== 'auto') return requested as any;
    if (wrap && wrap.closest('.el-dialog__body')) return 'container';
    return 'viewport';
  }

  function computeOnce() {
    const wrap = wrapRef.value;
    if (!wrap) return;
    if (options.mode === 'none') {
      height.value = 0;
      return;
    }

    const mode = pickMode();
    const rect = wrap.getBoundingClientRect();

    if (mode === 'container') {
      const bySelector = options.containerSelector
        ? (wrap.closest(options.containerSelector) as HTMLElement) || (document.querySelector(options.containerSelector) as HTMLElement)
        : null;
      containerEl = bySelector || (wrap.closest('.el-dialog__body') as HTMLElement) || wrap.parentElement;
      if (containerEl) {
        height.value = computeContainerAdaptiveHeight(rect.top, containerEl, options);
        return;
      }
      // Fall back to viewport mode when no container can be resolved.
    }

    height.value = computeViewportAdaptiveHeight(wrap, rect.top, options);
  }

  function recompute() {
    if (rafId != null) cancelAnimationFrame(rafId);
    rafId = requestAnimationFrame(() => {
      rafId = null;
      computeOnce();
    });
  }

  function setupObservers() {
    if (options.mode === 'none') return;
    // Minimal FE test DOM may omit window event APIs; skip rather than throw.
    if (typeof window.addEventListener === 'function') {
      window.addEventListener('resize', recompute);
      window.addEventListener('orientationchange', recompute);
    }

    const targets = collectAdaptiveHeightObserveTargets(wrapRef.value, options);
    const RO = typeof ResizeObserver === 'function' ? ResizeObserver : null;
    if (!targets.length || !RO) return;
    containerEl = targets[0] ?? null;
    ro = new RO(() => recompute());
    for (const el of targets) {
      ro.observe(el);
      observedEls.push(el);
    }
  }

  function cleanupObservers() {
    if (typeof window.removeEventListener === 'function') {
      window.removeEventListener('resize', recompute);
      window.removeEventListener('orientationchange', recompute);
    }
    if (ro) {
      for (const el of observedEls) {
        ro.unobserve(el);
      }
    }
    observedEls = [];
    ro = null;
    containerEl = null;
  }

  onMounted(() => {
    // Compute once after the first frame.
    recompute();
    setupObservers();
    // Recompute one more frame later to absorb follow-up layout shifts.
    requestAnimationFrame(() => recompute());
  });
  // Recompute when a KeepAlive view is activated again.
  onActivated(() => {
    recompute();
    // Run one more pass on the next frame after activation.
    requestAnimationFrame(() => recompute());
  });
  onBeforeUnmount(() => {
    cleanupObservers();
  });

  return { height, pxHeight, recompute };
}
