// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Component } from 'vue';
import type { RouteLocationNormalized, RouteRecordNormalized } from 'vue-router';

/**
 * Breadcrumb navigation item.
 * Defines the data structure used by breadcrumb navigation.
 */
export interface BreadcrumbItem {
  /**
   * Breadcrumb item path.
   */
  path: string;

  /**
   * Breadcrumb item title.
   */
  title: string;

  /**
   * Breadcrumb item icon.
   */
  icon?: Component;

  /**
   * Whether the breadcrumb item is disabled.
   * @default false
   */
  disabled?: boolean;

  /**
   * Route query parameters.
   */
  query?: Record<string, string>;
}

/**
 * Breadcrumb navigation action.
 * Describes the pure data structure used for breadcrumb navigation actions.
 */
export interface BreadcrumbNavigationAction {
  /**
   * Whether navigation is allowed.
   */
  navigable: boolean;

  /**
   * Target path.
   */
  path: string;

  /**
   * Route query parameters.
   */
  query?: Record<string, string>;
}
