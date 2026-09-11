// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineAction, defineModelActions } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt: listLt } = createTranslate('partner', { scope: 'web/views/PartnerListView' });
const { _lt: formLt } = createTranslate('partner', { scope: 'web/views/PartnerFormView' });

/** Action used to open partner detail from the list. */
export const partnerOpenDetailAction = defineAction('partner.action.partner_open_detail', {
  title: listLt('Open Partner Detail'),
  requires: [{ model: 'partner.Partner' }],
});

/** Standard CRUD action ids for partner.Partner list/form views. */
export const partnerActions = defineModelActions('partner.Partner', {
  entityTitle: formLt('Partner'),
});
