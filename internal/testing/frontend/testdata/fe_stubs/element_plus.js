// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * FE unit stub for `element-plus` (not the real package).
 * Any `El*` named import resolves to a passthrough slot stub via Proxy exports.
 */
import { makeSlotStub } from './slot_stub.js';

var cache = Object.create(null);
function stub(name) {
  if (!cache[name]) cache[name] = makeSlotStub(name);
  return cache[name];
}

export var ElMessage = {
  success: function () {},
  error: function () {},
  warning: function () {},
  info: function () {},
};
export var ElMessageBox = {
  confirm: function () {
    return Promise.resolve();
  },
  alert: function () {
    return Promise.resolve();
  },
};

// esbuild named-import interop: enumerate common components + Proxy for anything else via default.
var names = [
  'ElForm', 'ElFormItem', 'ElInput', 'ElButton', 'ElCheckbox', 'ElAlert', 'ElCard', 'ElIcon',
  'ElRow', 'ElCol', 'ElSelect', 'ElOption', 'ElDialog', 'ElTable', 'ElTableColumn', 'ElTag',
  'ElSwitch', 'ElTooltip', 'ElDropdown', 'ElDropdownMenu', 'ElDropdownItem', 'ElMenu',
  'ElMenuItem', 'ElSubMenu', 'ElPagination', 'ElLoading', 'ElDivider', 'ElSkeleton',
  'ElButtonGroup', 'ElTabs', 'ElTabPane', 'ElRadio', 'ElRadioGroup', 'ElRadioButton',
  'ElPopover', 'ElDrawer', 'ElScrollbar', 'ElEmpty', 'ElProgress', 'ElBadge', 'ElAvatar',
  'ElSpace', 'ElLink', 'ElText', 'ElUpload', 'ElDatePicker', 'ElTimePicker', 'ElTree',
  'ElCascader', 'ElAutocomplete', 'ElInputNumber', 'ElSlider', 'ElRate', 'ElColorPicker',
  'ElTransfer', 'ElCalendar', 'ElImage', 'ElBacktop', 'ElPageHeader', 'ElSteps', 'ElStep',
  'ElTimeline', 'ElTimelineItem', 'ElCollapse', 'ElCollapseItem', 'ElCarousel', 'ElCarouselItem',
  'ElResult', 'ElStatistic', 'ElAffix', 'ElAnchor', 'ElCheckTag', 'ElContainer', 'ElHeader',
  'ElAside', 'ElMain', 'ElFooter', 'ElConfigProvider',
];

var exportsObj = {
  ElMessage: ElMessage,
  ElMessageBox: ElMessageBox,
  default: {
    install: function (app) {
      for (var i = 0; i < names.length; i++) {
        app.component(names[i], stub(names[i]));
      }
    },
  },
};
for (var i = 0; i < names.length; i++) {
  exportsObj[names[i]] = stub(names[i]);
}

export var ElForm = exportsObj.ElForm;
export var ElFormItem = exportsObj.ElFormItem;
export var ElInput = exportsObj.ElInput;
export var ElButton = exportsObj.ElButton;
export var ElCheckbox = exportsObj.ElCheckbox;
export var ElAlert = exportsObj.ElAlert;
export var ElCard = exportsObj.ElCard;
export var ElIcon = exportsObj.ElIcon;
export var ElRow = exportsObj.ElRow;
export var ElCol = exportsObj.ElCol;
export var ElSelect = exportsObj.ElSelect;
export var ElOption = exportsObj.ElOption;
export var ElDialog = exportsObj.ElDialog;
export var ElTable = exportsObj.ElTable;
export var ElTableColumn = exportsObj.ElTableColumn;
export var ElTag = exportsObj.ElTag;
export var ElSwitch = exportsObj.ElSwitch;
export var ElTooltip = exportsObj.ElTooltip;
export var ElDropdown = exportsObj.ElDropdown;
export var ElDropdownMenu = exportsObj.ElDropdownMenu;
export var ElDropdownItem = exportsObj.ElDropdownItem;
export var ElMenu = exportsObj.ElMenu;
export var ElMenuItem = exportsObj.ElMenuItem;
export var ElSubMenu = exportsObj.ElSubMenu;
export var ElPagination = exportsObj.ElPagination;
export var ElLoading = exportsObj.ElLoading;
export var ElDivider = exportsObj.ElDivider;
export var ElSkeleton = exportsObj.ElSkeleton;
export var ElButtonGroup = exportsObj.ElButtonGroup;
export var ElTabs = exportsObj.ElTabs;
export var ElTabPane = exportsObj.ElTabPane;
export var ElRadio = exportsObj.ElRadio;
export var ElRadioGroup = exportsObj.ElRadioGroup;
export var ElRadioButton = exportsObj.ElRadioButton;
export var ElPopover = exportsObj.ElPopover;
export var ElDrawer = exportsObj.ElDrawer;
export var ElScrollbar = exportsObj.ElScrollbar;
export var ElEmpty = exportsObj.ElEmpty;
export var ElProgress = exportsObj.ElProgress;
export var ElBadge = exportsObj.ElBadge;
export var ElAvatar = exportsObj.ElAvatar;
export var ElSpace = exportsObj.ElSpace;
export var ElLink = exportsObj.ElLink;
export var ElUpload = exportsObj.ElUpload;
export var ElDatePicker = exportsObj.ElDatePicker;
export var ElTimePicker = exportsObj.ElTimePicker;
export var ElConfigProvider = exportsObj.ElConfigProvider;

export default exportsObj.default;
