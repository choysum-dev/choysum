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
export { default as ChoyListView } from './components/view/ChoyListView.vue';
export { default as ChoySearchView } from './components/view/ChoySearchView.vue';
export { default as ChoyPagination } from './components/view/ChoyPagination.vue';
export { default as ChoyBreadcrumb } from './components/view/ChoyBreadcrumb.vue';
export type { ChoyBreadcrumbItem } from './components/view/ChoyBreadcrumb.vue';
export {
  buildChoySearchQuery,
  filterRowsByKeyword,
  normalizeChoySearchKeyword,
  type ChoySearchFilter,
  type ChoySearchQuery,
} from './components/view/searchViewHelpers';
export {
  clampChoyPage,
  choyPageOffset,
  choyTotalPages,
} from './components/view/paginationHelpers';

export { default as ChoyFieldBase } from './components/field/ChoyFieldBase.vue';
export { default as ChoyVarcharField } from './components/field/ChoyVarcharField.vue';
export { default as ChoyTextField } from './components/field/ChoyTextField.vue';
export { default as ChoyNumberField } from './components/field/ChoyNumberField.vue';
export { default as ChoyMonetaryField } from './components/field/ChoyMonetaryField.vue';
export { default as ChoyBooleanField } from './components/field/ChoyBooleanField.vue';
export { default as ChoySelectionField } from './components/field/ChoySelectionField.vue';
export { default as ChoyStatusbarField } from './components/field/ChoyStatusbarField.vue';
export { default as ChoyDateField } from './components/field/ChoyDateField.vue';
export { default as ChoyDatetimeField } from './components/field/ChoyDatetimeField.vue';
export { default as ChoyTimeField } from './components/field/ChoyTimeField.vue';
export { default as ChoyManyToOneField } from './components/field/ChoyManyToOneField.vue';
export { default as ChoyBinaryField } from './components/field/ChoyBinaryField.vue';
export { default as ChoyImageField } from './components/field/ChoyImageField.vue';
export { default as ChoyVirtualField } from './components/field/ChoyVirtualField.vue';
export { default as ChoyFieldTranslationsDialog } from './components/field/ChoyFieldTranslationsDialog.vue';
export { default as ChoyFieldCompanyValuesDialog } from './components/field/ChoyFieldCompanyValuesDialog.vue';
export {
  formatChoyMonetary,
  parseChoyNumber,
  resolveChoyFieldVisible,
  resolveChoyMonetaryPrecision,
  resolveChoyNumberDraftText,
  roundChoyDecimal,
  type ChoyFieldChromeProps,
  type ChoySelectionOption,
} from './components/field/fieldHelpers';
export type { ChoyBinaryValue } from './components/field/ChoyBinaryField.vue';
export type { ChoyImageValue } from './components/field/ChoyImageField.vue';
export type { ChoyTranslationRow } from './components/field/ChoyFieldTranslationsDialog.vue';
export type { ChoyCompanyValueRow } from './components/field/ChoyFieldCompanyValuesDialog.vue';

export { ChoyMessage, useChoyMessage } from './composables/useChoyMessage';
export type { ChoyMessageLevel, ChoyMessageOptions } from './composables/useChoyMessage';
