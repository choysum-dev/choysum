// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Sidebar display mode.
 * - expanded: sidebar is fully expanded
 * - hover: sidebar expands temporarily while hovered
 * - collapsed: only icons remain visible
 * - hidden: sidebar is fully hidden
 */
export type SidebarMode = 'expanded' | 'collapsed' | 'hidden' | 'hover';

/**
 * Device type.
 */
export type DeviceType = 'mobile' | 'tablet' | 'desktop';

/**
 * Layout preference contract.
 */
export interface LayoutPreference {
  mode: SidebarMode;
  deviceType: DeviceType;
}
