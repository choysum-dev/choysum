// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineStore } from 'pinia';
import { ref, computed, watch } from 'vue';
import { useBreakpoints, isClient, useCssVar } from '@vueuse/core';
import type { DeviceType, SidebarMode, LayoutPreference } from './types';

export * from './types';

/**
 * Layout state store.
 * Manages sidebar state, responsive breakpoints, and persisted layout preferences.
 */
export const useLayoutStore = defineStore(
  'choysum.web.layout',
  () => {
    // ==================== Responsive breakpoints ====================

    // Breakpoints are only active on the client.
    const breakpoints = isClient
      ? useBreakpoints({
          xs: 0,
          sm: 640,
          md: 768,
          lg: 1024,
          xl: 1280,
          '2xl': 1536,
        })
      : {
          smallerOrEqual: () => ref(false),
          between: () => ref(false),
          greaterOrEqual: () => ref(true), // Treat SSR as desktop by default.
        };

    // Derive the current device type from one source of truth.
    const deviceType = computed<DeviceType>(() => {
      // Default SSR rendering to desktop.
      if (!isClient) return 'desktop';

      // Compute the device type from client-side breakpoints.
      if (breakpoints.smallerOrEqual('md').value) return 'mobile';
      if (breakpoints.between('md', 'lg').value) return 'tablet';
      return 'desktop';
    });

    // Convenience device flags.
    const isMobile = computed(() => deviceType.value === 'mobile');
    const isTablet = computed(() => deviceType.value === 'tablet');
    const isDesktop = computed(() => deviceType.value === 'desktop');

    // ==================== Sidebar state ====================

    // Persisted layout preference.
    const layoutPreference = ref<LayoutPreference | null>(null);

    // Derive sidebar mode from either persisted preference or current device type.
    const sidebarMode = computed<SidebarMode>({
      get() {
        // Return a stable default during SSR.
        if (!isClient) return 'expanded';

        // Reuse the saved preference only when it matches the current device class.
        if (layoutPreference.value && layoutPreference.value.deviceType === deviceType.value) {
          return layoutPreference.value.mode;
        }

        // Defaults: hidden on mobile, collapsed on tablet, expanded on desktop.
        if (isMobile.value) return 'hidden';
        if (isTablet.value) return 'collapsed';
        return 'expanded';
      },
      set(newMode: SidebarMode) {
        // Persist user-selected sidebar modes automatically.
        if (isClient) {
          layoutPreference.value = {
            mode: newMode,
            deviceType: deviceType.value,
          };
        }
      },
    });

    // Sets the sidebar mode, optionally treating it as a user preference update.
    function setSidebarMode(mode: SidebarMode, options?: { isUserAction?: boolean }) {
      // Skip mutations during SSR.
      if (!isClient) return;

      const isUserAction = options?.isUserAction ?? false;

      // User actions persist through the computed setter.
      if (isUserAction) {
        sidebarMode.value = mode;
      } else {
        // Non-user changes stay temporary and do not overwrite preferences.
        if (isClient) {
          const temp = layoutPreference.value;
          layoutPreference.value = null;
          sidebarMode.value = mode;
          layoutPreference.value = temp;
        }
      }
    }

    // Toggle the sidebar through the supported mode sequence.
    function toggleSidebar() {
      // Skip mutations during SSR.
      if (!isClient) return;

      switch (sidebarMode.value) {
        case 'expanded':
          setSidebarMode('collapsed', { isUserAction: true });
          break;
        case 'collapsed':
          setSidebarMode(isMobile.value ? 'hidden' : 'expanded', { isUserAction: true });
          break;
        case 'hidden':
          setSidebarMode('expanded', { isUserAction: true });
          break;
      }
    }

    // Hide the sidebar on mobile views.
    function closeSidebar() {
      // Skip mutations during SSR.
      if (!isClient) return;

      if (isMobile.value && sidebarMode.value === 'expanded') {
        setSidebarMode('hidden', { isUserAction: true });
      }
    }

    // ==================== Screen-size reactions ====================

    // Reset the sidebar when the device class changes and no matching preference exists.
    if (isClient) {
      watch(deviceType, (newDeviceType, oldDeviceType) => {
        if (newDeviceType !== oldDeviceType) {
          // Revert to defaults when no matching preference exists for the new device class.
          if (!layoutPreference.value || layoutPreference.value.deviceType !== newDeviceType) {
            setSidebarMode(newDeviceType === 'mobile' ? 'hidden' : newDeviceType === 'tablet' ? 'collapsed' : 'expanded');
          }
        }
      });
    }

    // ==================== Public store API ====================
    return {
      // Core state.
      layoutPreference,
      sidebarMode,
      deviceType,

      // Convenience flags.
      isMobile,
      isTablet,
      isDesktop,

      // Actions.
      setSidebarMode,
      toggleSidebar,
      closeSidebar,
    };
  },
  {
    // Persist user layout preferences.
    persist: {
      key: 'choysum.web.layout',
      storage: localStorage,
      pick: ['layoutPreference'],
    },
  }
);
