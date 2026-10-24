// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Compute, Decimal, Field, Model } from '@/core/service';
import type Currency from '@/base/service/models/currency';
import type Company from '@/base/service/models/company';
import { _lt } from '../i18n';
import SandboxCategory from './sandbox_category';
import SandboxLine from './sandbox_line';
import SandboxRecordCategory from './sandbox_record_category';
import SandboxRecordTag from './sandbox_record_tag';
import SandboxRecordTagChip from './sandbox_record_tag_chip';
import SandboxTag from './sandbox_tag';


const RECORD_STATES = [
  { value: 'draft', label: 'Draft' },
  { value: 'open', label: 'Open' },
  { value: 'done', label: 'Done' },
] as const;

/**
 * Kitchen-sink record covering every ORM FieldType plus depth options
 * (translate, companyDependent, condition, Compute, related, widgets).
 */
@Model('SandboxRecord')
export default class SandboxRecord extends BaseModel {
  // --- identity / required ---
  @Field({
    type: 'char',
    size: 64,
    notNull: true,
    string: _lt('Char Name', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('Dedicated char column (not varchar) for FieldsGet/Search coverage.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  NameChar: string;

  @Field({
    type: 'varchar',
    size: 120,
    notNull: true,
    translate: true,
    index: 'trigram',
    string: _lt('Title', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('Translatable varchar; use the translation action on the field.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  Title: string;

  @Field({
    type: 'varchar',
    size: 40,
    notNull: true,
    unique: true,
    index: true,
    string: _lt('Code', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  Code: string;

  // --- text / html ---
  @Field({
    type: 'text',
    translate: true,
    string: _lt('Notes', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  Notes?: string;

  @Field({
    type: 'html',
    string: _lt('Body HTML', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  BodyHtml?: string;

  // --- numbers ---
  @Field({
    type: 'int',
    default: () => 0,
    string: _lt('Int Value', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  IntValue?: number;

  @Field({
    type: 'bigint',
    string: _lt('Bigint Value', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  BigintValue?: string | number;

  @Field({
    type: 'number',
    string: _lt('Number Value', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  NumberValue?: number;

  @Field({
    type: 'decimal',
    scale: 4,
    string: _lt('Decimal Value', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  DecimalValue?: Decimal;

  @Field<Currency>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Currency' },
    condition: ['IsActive', '=', true],
    size: 20,
    index: true,
    string: _lt('Currency', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  CurrencyId?: string;

  @Field({
    type: 'monetary',
    // Flat currencyField is accepted at runtime; public FieldOptions overload lags the decorator bag.
    // @ts-expect-error currencyField is validated against this model at Model() load
    currencyField: 'CurrencyId',
    string: _lt('Amount', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  Amount?: Decimal;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    string: _lt('Active', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  IsActive: boolean;

  @Field({
    type: 'int',
    notNull: true,
    default: () => 10,
    index: true,
    string: _lt('Sequence', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  Sequence: number;

  // --- temporal ---
  @Field({
    type: 'date',
    string: _lt('Date Value', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  DateValue?: Date | string;

  @Field({
    type: 'datetime',
    string: _lt('Datetime Value', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  DatetimeValue?: Date | string;

  @Field({
    type: 'time',
    string: _lt('Time Value', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  TimeValue?: Date | string;

  // --- choice ---
  @Field({
    type: 'selection',
    selection: () => [...RECORD_STATES],
    size: 16,
    notNull: true,
    default: () => 'draft',
    index: true,
    string: _lt('State', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  State: string;

  @Field({
    type: 'selection',
    selection: () => [...RECORD_STATES],
    size: 16,
    notNull: true,
    default: () => 'draft',
    string: _lt('Progress', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('Same selection options as State; Form binds ChoyStatusbarField.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  Progress: string;

  // --- binary / image / json / properties ---
  @Field({
    type: 'binary',
    string: _lt('Attachment', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  Attachment?: string;

  @Field({
    type: 'image',
    string: _lt('Image', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  Image?: string;

  @Field({
    type: 'jsonobject',
    string: _lt('JSON Payload', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  JsonPayload?: Record<string, unknown>;

  @Field({
    type: 'properties',
    string: _lt('Properties', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('App-level properties container (no parent definition field).', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  Properties?: Record<string, unknown>;

  // --- depth: company-dependent (not combined with translate) ---
  @Field({
    type: 'varchar',
    size: 200,
    companyDependent: true,
    string: _lt('Company Note', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('Per-company varchar; open Company Values from the field chrome.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  CompanyNote?: string;

  @Field<Company>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    condition: ['IsActive', '=', true],
    size: 20,
    index: true,
    string: _lt('Company', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  CompanyId?: string;

  // --- relations ---
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => SandboxCategory },
    condition: ['IsActive', '=', true],
    index: true,
    string: _lt('Category', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('Domain restricts to active categories only.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  CategoryId?: SandboxCategory;

  @Field({
    type: 'varchar',
    size: 100,
    related: { path: 'CategoryId.Name', store: false },
    string: _lt('Category Name', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  readonly CategoryName?: string;

  @Field<SandboxCategory>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'sandbox.SandboxCategory' },
    condition: ['IsActive', '=', true],
    size: 36,
    string: _lt('Category Ref', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  CategoryRefId?: string;

  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => SandboxRecordTag,
      targetModel: () => SandboxTag,
      joinField: 'RecordId',
      inverseJoinField: 'TagId',
    },
    condition: ['IsActive', '=', true],
    string: _lt('Tags', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  Tags?: SandboxTag[];

  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => SandboxRecordTagChip,
      targetModel: () => SandboxTag,
      joinField: 'RecordId',
      inverseJoinField: 'TagId',
    },
    condition: ['IsActive', '=', true],
    string: _lt('Tag Chips', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('Same-app ManyToMany bound with the Tags widget in Form.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  TagChips?: SandboxTag[];

  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => SandboxRecordCategory,
      targetModel: () => SandboxCategory,
      joinField: 'RecordId',
      inverseJoinField: 'CategoryId',
    },
    condition: ['IsActive', '=', true],
    string: _lt('Categories', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('Same-app ManyToMany bound with the Tree widget in Form.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  Categories?: SandboxCategory[];

  @Field<SandboxTag>({
    type: 'ManyToManyRef',
    relation: { targetModel: 'sandbox.SandboxTag' },
    string: _lt('Tag Refs', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  TagRefIds?: string[];

  @Field<SandboxCategory>({
    type: 'ManyToManyRef',
    relation: { targetModel: 'sandbox.SandboxCategory' },
    condition: ['IsActive', '=', true],
    string: _lt('Category Path', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('ManyToManyRef bound with the tree widget in Form.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  CategoryPathIds?: string[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => SandboxLine, inverseField: 'RecordId' },
    string: _lt('Lines', { scope: 'sandbox.model.SandboxRecord.fields' }),
  })
  Lines?: SandboxLine[];

  @Field({
    type: 'int',
    string: _lt('Line Count', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('Computed from Lines; not edited by hand.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  readonly LineCount?: number;

  @Compute<SandboxRecord>('LineCount', { deps: ['Lines.Id'], store: false })
  computeLineCount() {
    return (this.Lines || []).filter(line => !!line?.Id).length;
  }

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => SandboxLine },
    index: true,
    string: _lt('Primary Line', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('Computed: first line by Sequence then Id.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  readonly PrimaryLineId?: SandboxLine;

  @Compute<SandboxRecord>('PrimaryLineId', {
    deps: ['Lines.Id', 'Lines.Name', 'Lines.Sequence'],
    store: false,
  })
  computePrimaryLineId() {
    const lines = [...(this.Lines || [])]
      .filter(line => !!line?.Id)
      .sort((a, b) => {
        const seq = (a.Sequence ?? 0) - (b.Sequence ?? 0);
        if (seq !== 0) return seq;
        return String(a.Id).localeCompare(String(b.Id));
      });
    return lines[0];
  }
}
