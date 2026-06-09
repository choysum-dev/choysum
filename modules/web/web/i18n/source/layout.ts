// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Layout-related text in the English source locale.
 */
export default {
  appName: 'Choysum',

  // Sidebar.
  sidebar: {
    collapse: 'Collapse Sidebar',
    expand: 'Expand Sidebar',
    toggle: 'Toggle Sidebar',
    home: 'Home',
    dashboard: 'Dashboard',
    menu: 'Menu',
  },

  // Header.
  header: {
    menu: 'Menu',
    profile: 'Profile',
    settings: 'Settings',
    notifications: 'Notifications',
    languages: 'Languages',
    darkMode: 'Dark Mode',
    lightMode: 'Light Mode',
    search: 'Search...',
  },

  // Footer.
  footer: {
    copyright: '© {year} Choysum. All rights reserved.',
    version: 'Version {version}',
    powered: 'Powered by Choysum',
  },

  // Breadcrumb.
  breadcrumb: {
    home: 'Home',
  },

  // Page states.
  page: {
    loading: 'Loading page content...',
    notFound: 'Page not found',
    accessDenied: 'Access denied',
  },
} as const;
