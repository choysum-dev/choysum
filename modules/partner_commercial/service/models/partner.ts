// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field, Model } from '@/core/service';
import PartnerBase from '@/partner/service/models/partner';
import { _lt } from '../i18n';
import PartnerIdentifier from './partner_identifier';

/**
 * Partner extension that adds commercial identifier rows.
 */
@Model('Partner', { companyField: 'CompanyId' })
export default class Partner extends PartnerBase {
  /** Related commercial identifier rows. */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => PartnerIdentifier, inverseField: 'PartnerId' },
    string: _lt('Identifiers', { scope: 'partner_commercial.model.Partner.fields' }),
  })
  PartnerIdentifiers?: PartnerIdentifier[];
}
