// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export { webApiService } from './service';
export {
  beginNavigationLoading,
  getControlBusy,
  getSurfaceBusy,
  hasActiveSurfaceBusy,
  loadingTiming,
  resetLoadingRegistryForTests,
  runWithLoading,
  runWithLoadingSync,
  scheduleNavigationSettleIfIdle,
  settleNavigationIfActive,
  useControlBusy,
  useHasActiveSurfaceBusy,
  useLoadingBarVisible,
  useLoadingSlowHintVisible,
  useNavigationHandoffBusy,
  useSurfaceBusy,
  type ControlBusySnapshot,
  type LoadingLayer,
  type LoadingOpts,
  type SurfaceBusySnapshot,
  type SurfaceMode,
} from './loadingRegistry';
