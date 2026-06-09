// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';

@Model('Language')
export default class Language extends BaseModel {
  @Field({ type: 'varchar', column: { size: 100, unique: true, notNull: true } })
  Name: string;
}
