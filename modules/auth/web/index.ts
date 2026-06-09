// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Import extension components for their side effects.
 *
 * Importing OHeader.vue and App.vue activates their XPath-based UI extensions.
 */

// Import side-effect extensions that augment shared layout components.
import './components/layout/OHeader.vue';
import './App.vue';

// Import auth-scoped error helpers.
import './error';

// Re-export the auth module application entry.
export { default } from './app';
