// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { choyListPickerChromeProps } from './choyListPickerChrome';

describe('choyListPickerChromeProps', () => {
  test('single selection hides selection column and disables chrome', () => {
    const props = choyListPickerChromeProps({ selectionMode: 'single' });
    expect(props).toEqual({
      showActions: false,
      hostPageTitle: false,
      registerActionTarget: false,
      clickToSelect: true,
      heightMode: 'container',
      containerSelector: '.choy-relation-picker-list',
      selectionMode: 'single',
      detailPath: undefined,
      showSelectionColumn: false,
    });
  });

  test('multiple selection omits showSelectionColumn unless forced false', () => {
    const props = choyListPickerChromeProps({ selectionMode: 'multiple' });
    expect(props.showSelectionColumn).toBeUndefined();
    expect(props.selectionMode).toBe('multiple');
    expect(props.detailPath).toBeUndefined();
    expect(props.showActions).toBe(false);

    const forced = choyListPickerChromeProps({
      selectionMode: 'multiple',
      showSelectionColumn: false,
    });
    expect(forced.showSelectionColumn).toBe(false);
  });

  test('containerSelector override', () => {
    const props = choyListPickerChromeProps({
      selectionMode: 'single',
      containerSelector: '.custom-host',
    });
    expect(props.containerSelector).toBe('.custom-host');
  });
});
