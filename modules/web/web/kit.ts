// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Public Choy UI kit surface (no app instance).
 * Domain modules import Choy* names from `@/web` (re-exports this file).
 * Never import ui/*, internal/*, Reka, Unovis, or TanStack from domain code.
 */

export { default as ChoyLayout } from './components/layout/ChoyLayout.vue';
export { default as ChoyPage } from './components/layout/ChoyPage.vue';
export { default as ChoyPageIoMenu } from './components/layout/ChoyPageIoMenu.vue';
export { default as ChoyActionTray } from './components/layout/ChoyActionTray.vue';
export { default as ChoyConfirmHost } from './components/layout/ChoyConfirmHost.vue';
export { default as ChoySkeleton } from './components/layout/ChoySkeleton.vue';
export { default as ChoyBadge } from './components/layout/ChoyBadge.vue';
export { default as ChoyCard } from './components/layout/ChoyCard.vue';
export { default as ChoyGrid } from './components/layout/ChoyGrid.vue';
export { default as ChoyCol } from './components/layout/ChoyCol.vue';
export { default as ChoyTabs } from './components/layout/ChoyTabs.vue';
export { default as ChoyTab } from './components/layout/ChoyTab.vue';
export { default as ChoyButton } from './components/layout/ChoyButton.vue';
export { default as ChoyNotificationBell } from './components/layout/ChoyNotificationBell.vue';

export { default as ChoyFormView } from './components/view/ChoyFormView.vue';
export { default as ChoyListView } from './components/view/ChoyListView.vue';
export { default as ChoyKanbanView } from './components/view/ChoyKanbanView.vue';
export { default as ChoyChartView } from './components/view/ChoyChartView.vue';
export { default as ChoySearchView } from './components/view/ChoySearchView.vue';
export { default as ChoyPagination } from './components/view/ChoyPagination.vue';
export { default as ChoyBreadcrumb } from './components/view/ChoyBreadcrumb.vue';
export { default as ChoyViewScope } from './components/view/ChoyViewScope.vue';
export { default as ChoyButtonBox } from './components/view/ChoyButtonBox.vue';
export { default as ChoyStatInfo } from './components/view/ChoyStatInfo.vue';
export { default as ChoyTableColumn } from './components/table/ChoyTableColumn.vue';
export type { ChoyBreadcrumbItem } from './components/view/ChoyBreadcrumb.vue';
export type { ViewMode as ChoyViewMode, ViewContainer as ChoyViewContainer } from './components/view/ChoyViewScope.vue';
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
export {
  applyChoyKanbanMove,
  choyKanbanLaneRemain,
  formatChoyKanbanLoadMoreLabel,
  groupRowsIntoChoyKanbanLanes,
  normalizeChoyKanbanLaneKey,
  resolveChoyKanbanCardId,
  type ChoyKanbanCard,
  type ChoyKanbanLane,
  type ChoyKanbanLoadMore,
  type ChoyKanbanMove,
} from './components/view/kanbanViewHelpers';
export {
  createLaneSyncGate,
  finishInitialKanbanLoad,
  shouldRecoverStaleKanbanSearch,
  shouldRestoreKanbanMove,
} from './components/view/kanbanStoreHelpers';
export { useChoyKanbanStoreEngine } from './composables/useChoyKanbanStoreEngine';
export type { ChoyKanbanStoreEngineOptions } from './composables/useChoyKanbanStoreEngine';
export {
  availableChartTypes,
  resolveChartAdapter,
  chartTypeRegistry,
  CHOY_CHART_DEFAULT_PALETTE,
} from './components/view/chart/chartTypeAdapter';
export type {
  ChartBuildContext,
  ChartSupportContext,
  ChoyChartKind,
  ChoyChartSeries,
  ChoyChartSort,
  ChoyChartSpec,
  IChartTypeAdapter,
} from './components/view/chart/chartTypeAdapter';
export {
  chartSpecToPieRows,
  chartSpecToXyRows,
  normalizeSeriesToPercent,
  sortChartCategories,
} from './components/view/chartViewHelpers';
export type {
  ChoyChartItemClickPayload,
  ChoyChartMetricOption,
} from './components/view/chartViewHelpers';
export { groupRowsToChartSeries } from './components/view/chartStoreHelpers';

export { default as ChoyFieldBase } from './components/field/ChoyFieldBase.vue';
export { default as ChoyVarcharField } from './components/field/ChoyVarcharField.vue';
export { default as ChoyTextField } from './components/field/ChoyTextField.vue';
export { default as ChoyIntField } from './components/field/ChoyIntField.vue';
export { default as ChoyBigintField } from './components/field/ChoyBigintField.vue';
export { default as ChoyNumberField } from './components/field/ChoyNumberField.vue';
export { default as ChoyDecimalField } from './components/field/ChoyDecimalField.vue';
export { default as ChoyMonetaryField } from './components/field/ChoyMonetaryField.vue';
export { default as ChoyBooleanField } from './components/field/ChoyBooleanField.vue';
export { default as ChoySelectionField } from './components/field/ChoySelectionField.vue';
export { default as ChoyStatusbarField } from './components/field/ChoyStatusbarField.vue';
export { default as ChoyDateField } from './components/field/ChoyDateField.vue';
export { default as ChoyDatetimeField } from './components/field/ChoyDatetimeField.vue';
export { default as ChoyTimeField } from './components/field/ChoyTimeField.vue';
export { default as ChoyManyToOneField } from './components/field/ChoyManyToOneField.vue';
export { default as ChoyManyToOneRefField } from './components/field/ChoyManyToOneRefField.vue';
export { default as ChoyManyToManyField } from './components/field/ChoyManyToManyField.vue';
export { default as ChoyManyToManyRefTagsField } from './components/field/ChoyManyToManyRefTagsField.vue';
export { default as ChoyManyToManyRefTreeField } from './components/field/ChoyManyToManyRefTreeField.vue';
export { default as ChoyOneToManyField } from './components/field/ChoyOneToManyField.vue';
export { default as ChoyOneToManyKanbanField } from './components/field/ChoyOneToManyKanbanField.vue';
export { default as ChoyBinaryField } from './components/field/ChoyBinaryField.vue';
export { default as ChoyImageField } from './components/field/ChoyImageField.vue';
export { default as ChoyHtmlField } from './components/field/ChoyHtmlField.vue';
export { default as ChoyJsonField } from './components/field/ChoyJsonField.vue';
export { default as ChoyPropertiesField } from './components/field/ChoyPropertiesField.vue';
export { default as ChoyPropertiesDefinitionEditor } from './components/field/ChoyPropertiesDefinitionEditor.vue';
export { default as ChoyVirtualField } from './components/field/ChoyVirtualField.vue';
export { default as ChoyFieldTranslationsDialog } from './components/field/ChoyFieldTranslationsDialog.vue';
export { default as ChoyFieldCompanyValuesDialog } from './components/field/ChoyFieldCompanyValuesDialog.vue';
export {
  choyFieldChromeDefaults,
  formatChoyMonetary,
  parseChoyNumber,
  resolveChoyFieldVisible,
  resolveChoyMonetaryPrecision,
  resolveChoyNumberDraftText,
  roundChoyDecimal,
  type ChoyFieldChromeProps,
  type ChoySelectionOption,
} from './components/field/fieldHelpers';
export {
  sanitizeHtmlForClient,
  htmlToPlaintext,
  normalizeHtmlForStore,
} from './components/field/htmlHelpers';
export {
  normalizeChoyJsonIncoming,
  stringifyChoyJson,
  tryParseChoyJson,
  type ChoyJsonValue,
} from './components/field/jsonFieldHelpers';
export {
  buildFullPropertiesMap,
  filterRenderablePropertyItems,
  writePropertyValue,
  type PropertiesMap,
} from './components/field/propertiesHelpers';
export type { ChoyBinaryValue } from './components/field/fieldHelpers';
export type { ChoyImageValue } from './components/field/fieldHelpers';
export type { ChoyTranslationRow } from './components/field/ChoyFieldTranslationsDialog.vue';
export type { ChoyCompanyValueRow } from './components/field/ChoyFieldCompanyValuesDialog.vue';
export type {
  ChoyManyToManyWidget,
  ChoyManyToManyTreeNode,
  ChoyOneToManyWidget,
} from './components/field/choyRelationFieldTypes';

export { default as ChoyChatter } from './components/chatter/ChoyChatter.vue';
export { default as ChoyChatterComposer } from './components/chatter/ChoyChatterComposer.vue';
export { default as ChoyChatterTimeline } from './components/chatter/ChoyChatterTimeline.vue';
export { default as ChoyChatterFollowerBar } from './components/chatter/ChoyChatterFollowerBar.vue';
export { default as ChoyChatterMessageItem } from './components/chatter/ChoyChatterMessageItem.vue';
export { default as ChoyChatterFieldChangeItem } from './components/chatter/ChoyChatterFieldChangeItem.vue';
export {
  formatChoyUtcIso,
  formatFieldChangeSummary,
  resolveChoyChatterAuthorLabel,
} from './components/chatter/chatterHelpers';
export {
  compareChatterTimelineEntries,
  mergeChatterTimeline,
  parseChatterTimestamp,
} from './components/chatter/mergeChatterTimeline';
export type {
  ChatterFieldChangeEntry,
  ChatterFieldChangeRow,
  ChatterMessageEntry,
  ChatterMessageRow,
  ChatterTimelineEntry,
} from './components/chatter/chatterTypes';
export {
  PARTNER_DETAIL_TAB_PANELS_ANCHOR,
  PARTNER_DETAIL_TAB_PANELS_XPATH,
} from './pages/partnerDetailXpath';

export { ChoyMessage, useChoyMessage } from './composables/useChoyMessage';
export type { ChoyMessageLevel, ChoyMessageOptions } from './composables/useChoyMessage';
export {
  confirmChoyAction,
  confirmChoyChoice,
  resolveConfirmChoy,
  useConfirmChoyStore,
} from './composables/confirmChoyAction';
export type { ConfirmChoyChoice, ConfirmChoyOptions } from './composables/confirmChoyAction';
export { choyControlHeightPx } from './lib/choyControlHeight';
export {
  applyChoyThemePreference,
  persistChoyThemePreference,
  readChoyThemePreference,
  resolveChoyThemePreference,
  CHOY_THEME_STORAGE_KEY,
} from './composables/applyChoyThemePreference';
export type {
  ApplyChoyThemePreferenceOptions,
  ChoyDensityPreference,
  ChoyThemeMode,
  ChoyThemePreference,
  ResolvedChoyThemePreference,
} from './composables/applyChoyThemePreference';
