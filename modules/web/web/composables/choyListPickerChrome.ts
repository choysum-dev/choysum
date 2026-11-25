// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Shared ListView chrome for relation-picker dialogs (M2O / M2M / Ref / Tags).
 * Disables page actions, title host, action-target registration, and detail
 * navigation (via canOpenDetail); sizes the list to the dialog container.
 */

export type ChoyListPickerSelectionMode = 'single' | 'multiple';

export type ChoyListPickerChromeOptions = {
  selectionMode: ChoyListPickerSelectionMode;
  /** CSS selector for height-mode=container (picker list host). */
  containerSelector?: string;
  /** M2O pickers hide the selection column; M2M may leave default. */
  showSelectionColumn?: boolean;
};

export type ChoyListPickerChromeProps = {
  showActions: false;
  hostPageTitle: false;
  registerActionTarget: false;
  clickToSelect: true;
  heightMode: 'container';
  containerSelector: string;
  selectionMode: ChoyListPickerSelectionMode;
  /**
   * Domain ListViews often set a static detail-path after v-bind="$attrs", which
   * overrides an undefined detailPath. canOpenDetail false still blocks navigation.
   */
  canOpenDetail: false;
  showSelectionColumn?: false;
};

const DEFAULT_CONTAINER = '.choy-relation-picker-list';

/** Props to `v-bind` onto the dialog search/list view component. */
export function choyListPickerChromeProps(
  opts: ChoyListPickerChromeOptions,
): ChoyListPickerChromeProps {
  const props: ChoyListPickerChromeProps = {
    showActions: false,
    hostPageTitle: false,
    registerActionTarget: false,
    clickToSelect: true,
    heightMode: 'container',
    containerSelector: opts.containerSelector || DEFAULT_CONTAINER,
    selectionMode: opts.selectionMode,
    canOpenDetail: false,
  };
  const hideSelectionColumn =
    opts.showSelectionColumn === false ||
    (opts.showSelectionColumn == null && opts.selectionMode === 'single');
  if (hideSelectionColumn) {
    props.showSelectionColumn = false;
  }
  return props;
}
