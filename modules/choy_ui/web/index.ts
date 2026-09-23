// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Choy UI web entry (gallery / kit isolation).
 * Public L1 exports for dogfood and (later) cutover into modules/web.
 */
export { default } from './app';
export { registerChoyGalleryRoute, setupRouter, choyUiRoutes } from './route';

export { default as ChoyLayout } from './components/layout/ChoyLayout.vue';
export { default as ChoyPage } from './components/layout/ChoyPage.vue';
export { default as ChoyPageIoMenu } from './components/layout/ChoyPageIoMenu.vue';
export { default as ChoyCard } from './components/layout/ChoyCard.vue';
export { default as ChoyGrid } from './components/layout/ChoyGrid.vue';
export { default as ChoyCol } from './components/layout/ChoyCol.vue';
export { default as ChoyTabs } from './components/layout/ChoyTabs.vue';
export { default as ChoyTab } from './components/layout/ChoyTab.vue';
export { default as ChoyButton } from './components/layout/ChoyButton.vue';
export { default as ChoyNotificationBell } from './components/layout/ChoyNotificationBell.vue';
export { default as ChoyFormView } from './components/view/ChoyFormView.vue';
export { ChoyMessage, useChoyMessage } from './composables/useChoyMessage';
export type { ChoyMessageLevel, ChoyMessageOptions } from './composables/useChoyMessage';
