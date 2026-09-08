// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * FE unit stub for `element-plus` (not the real package).
 * Named exports cover El* imports used by modules; keep in sync with repo imports.
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

export var ElAffix = stub('ElAffix');
export var ElAlert = stub('ElAlert');
export var ElAnchor = stub('ElAnchor');
export var ElAside = stub('ElAside');
export var ElAutoResizer = stub('ElAutoResizer');
export var ElAutocomplete = stub('ElAutocomplete');
export var ElAvatar = stub('ElAvatar');
export var ElBacktop = stub('ElBacktop');
export var ElBadge = stub('ElBadge');
export var ElBreadcrumb = stub('ElBreadcrumb');
export var ElBreadcrumbItem = stub('ElBreadcrumbItem');
export var ElButton = stub('ElButton');
export var ElButtonGroup = stub('ElButtonGroup');
export var ElCalendar = stub('ElCalendar');
export var ElCard = stub('ElCard');
export var ElCarousel = stub('ElCarousel');
export var ElCarouselItem = stub('ElCarouselItem');
export var ElCascader = stub('ElCascader');
export var ElCheckTag = stub('ElCheckTag');
export var ElCheckbox = stub('ElCheckbox');
export var ElCol = stub('ElCol');
export var ElCollapse = stub('ElCollapse');
export var ElCollapseItem = stub('ElCollapseItem');
export var ElColorPicker = stub('ElColorPicker');
export var ElConfigProvider = stub('ElConfigProvider');
export var ElContainer = stub('ElContainer');
export var ElDatePicker = stub('ElDatePicker');
export var ElDescriptions = stub('ElDescriptions');
export var ElDescriptionsItem = stub('ElDescriptionsItem');
export var ElDialog = stub('ElDialog');
export var ElDivider = stub('ElDivider');
export var ElDrawer = stub('ElDrawer');
export var ElDropdown = stub('ElDropdown');
export var ElDropdownItem = stub('ElDropdownItem');
export var ElDropdownMenu = stub('ElDropdownMenu');
export var ElEmpty = stub('ElEmpty');
export var ElFooter = stub('ElFooter');
export var ElForm = stub('ElForm');
export var ElFormItem = stub('ElFormItem');
export var ElHeader = stub('ElHeader');
export var ElIcon = stub('ElIcon');
export var ElImage = stub('ElImage');
export var ElInput = stub('ElInput');
export var ElInputNumber = stub('ElInputNumber');
export var ElLink = stub('ElLink');
export var ElLoading = stub('ElLoading');
export var ElMain = stub('ElMain');
export var ElMenu = stub('ElMenu');
export var ElMenuItem = stub('ElMenuItem');
export var ElOption = stub('ElOption');
export var ElOverlay = stub('ElOverlay');
export var ElPageHeader = stub('ElPageHeader');
export var ElPagination = stub('ElPagination');
export var ElPopover = stub('ElPopover');
export var ElProgress = stub('ElProgress');
export var ElRadio = stub('ElRadio');
export var ElRadioButton = stub('ElRadioButton');
export var ElRadioGroup = stub('ElRadioGroup');
export var ElRate = stub('ElRate');
export var ElResult = stub('ElResult');
export var ElRow = stub('ElRow');
export var ElScrollbar = stub('ElScrollbar');
export var ElSegmented = stub('ElSegmented');
export var ElSelect = stub('ElSelect');
export var ElSelectV2 = stub('ElSelectV2');
export var ElSkeleton = stub('ElSkeleton');
export var ElSlider = stub('ElSlider');
export var ElSpace = stub('ElSpace');
export var ElStatistic = stub('ElStatistic');
export var ElStep = stub('ElStep');
export var ElSteps = stub('ElSteps');
export var ElSubMenu = stub('ElSubMenu');
export var ElSwitch = stub('ElSwitch');
export var ElTabPane = stub('ElTabPane');
export var ElTable = stub('ElTable');
export var ElTableColumn = stub('ElTableColumn');
export var ElTableV2 = stub('ElTableV2');
export var ElTabs = stub('ElTabs');
export var ElTag = stub('ElTag');
export var ElText = stub('ElText');
export var ElTimePicker = stub('ElTimePicker');
export var ElTimeSelect = stub('ElTimeSelect');
export var ElTimeline = stub('ElTimeline');
export var ElTimelineItem = stub('ElTimelineItem');
export var ElTooltip = stub('ElTooltip');
export var ElTransfer = stub('ElTransfer');
export var ElTree = stub('ElTree');
export var ElTreeSelect = stub('ElTreeSelect');
export var ElTreeV2 = stub('ElTreeV2');
export var ElUpload = stub('ElUpload');

export default {
  install: function (app) {
    app.component('ElAffix', ElAffix);
    app.component('ElAlert', ElAlert);
    app.component('ElAnchor', ElAnchor);
    app.component('ElAside', ElAside);
    app.component('ElAutoResizer', ElAutoResizer);
    app.component('ElAutocomplete', ElAutocomplete);
    app.component('ElAvatar', ElAvatar);
    app.component('ElBacktop', ElBacktop);
    app.component('ElBadge', ElBadge);
    app.component('ElBreadcrumb', ElBreadcrumb);
    app.component('ElBreadcrumbItem', ElBreadcrumbItem);
    app.component('ElButton', ElButton);
    app.component('ElButtonGroup', ElButtonGroup);
    app.component('ElCalendar', ElCalendar);
    app.component('ElCard', ElCard);
    app.component('ElCarousel', ElCarousel);
    app.component('ElCarouselItem', ElCarouselItem);
    app.component('ElCascader', ElCascader);
    app.component('ElCheckTag', ElCheckTag);
    app.component('ElCheckbox', ElCheckbox);
    app.component('ElCol', ElCol);
    app.component('ElCollapse', ElCollapse);
    app.component('ElCollapseItem', ElCollapseItem);
    app.component('ElColorPicker', ElColorPicker);
    app.component('ElConfigProvider', ElConfigProvider);
    app.component('ElContainer', ElContainer);
    app.component('ElDatePicker', ElDatePicker);
    app.component('ElDescriptions', ElDescriptions);
    app.component('ElDescriptionsItem', ElDescriptionsItem);
    app.component('ElDialog', ElDialog);
    app.component('ElDivider', ElDivider);
    app.component('ElDrawer', ElDrawer);
    app.component('ElDropdown', ElDropdown);
    app.component('ElDropdownItem', ElDropdownItem);
    app.component('ElDropdownMenu', ElDropdownMenu);
    app.component('ElEmpty', ElEmpty);
    app.component('ElFooter', ElFooter);
    app.component('ElForm', ElForm);
    app.component('ElFormItem', ElFormItem);
    app.component('ElHeader', ElHeader);
    app.component('ElIcon', ElIcon);
    app.component('ElImage', ElImage);
    app.component('ElInput', ElInput);
    app.component('ElInputNumber', ElInputNumber);
    app.component('ElLink', ElLink);
    app.component('ElLoading', ElLoading);
    app.component('ElMain', ElMain);
    app.component('ElMenu', ElMenu);
    app.component('ElMenuItem', ElMenuItem);
    app.component('ElOption', ElOption);
    app.component('ElOverlay', ElOverlay);
    app.component('ElPageHeader', ElPageHeader);
    app.component('ElPagination', ElPagination);
    app.component('ElPopover', ElPopover);
    app.component('ElProgress', ElProgress);
    app.component('ElRadio', ElRadio);
    app.component('ElRadioButton', ElRadioButton);
    app.component('ElRadioGroup', ElRadioGroup);
    app.component('ElRate', ElRate);
    app.component('ElResult', ElResult);
    app.component('ElRow', ElRow);
    app.component('ElScrollbar', ElScrollbar);
    app.component('ElSegmented', ElSegmented);
    app.component('ElSelect', ElSelect);
    app.component('ElSelectV2', ElSelectV2);
    app.component('ElSkeleton', ElSkeleton);
    app.component('ElSlider', ElSlider);
    app.component('ElSpace', ElSpace);
    app.component('ElStatistic', ElStatistic);
    app.component('ElStep', ElStep);
    app.component('ElSteps', ElSteps);
    app.component('ElSubMenu', ElSubMenu);
    app.component('ElSwitch', ElSwitch);
    app.component('ElTabPane', ElTabPane);
    app.component('ElTable', ElTable);
    app.component('ElTableColumn', ElTableColumn);
    app.component('ElTableV2', ElTableV2);
    app.component('ElTabs', ElTabs);
    app.component('ElTag', ElTag);
    app.component('ElText', ElText);
    app.component('ElTimePicker', ElTimePicker);
    app.component('ElTimeSelect', ElTimeSelect);
    app.component('ElTimeline', ElTimeline);
    app.component('ElTimelineItem', ElTimelineItem);
    app.component('ElTooltip', ElTooltip);
    app.component('ElTransfer', ElTransfer);
    app.component('ElTree', ElTree);
    app.component('ElTreeSelect', ElTreeSelect);
    app.component('ElTreeV2', ElTreeV2);
    app.component('ElUpload', ElUpload);
  },
};
