// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Public Dialog chrome for kit-host product surfaces (Import / Search / Field).
 * Domain modules import the same names from `@/web` / kit.ts.
 *
 * Content is a product wrapper (dismissible); Footer is product layout sugar.
 * Root / Title / Description / Header stay vendor re-exports.
 */
export { default as ChoyDialog } from '../vendor/ui/dialog/Dialog.vue';
export { default as ChoyDialogContent } from './ChoyDialogContent.vue';
export { default as ChoyDialogFooter } from './ChoyDialogFooter.vue';
export { default as ChoyDialogDescription } from '../vendor/ui/dialog/DialogDescription.vue';
export { default as ChoyDialogHeader } from '../vendor/ui/dialog/DialogHeader.vue';
export { default as ChoyDialogTitle } from '../vendor/ui/dialog/DialogTitle.vue';
