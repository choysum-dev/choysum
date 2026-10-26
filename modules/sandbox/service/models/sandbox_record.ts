// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Compute, Decimal, Field, Model } from '@/core/service';
import { Onchange } from '@/core/service/api/onchange';
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

const RECORD_STATE_VALUES = RECORD_STATES.map(s => s.value);

/** CompanyNote prefix written by CategoryId onchange (cleared only when still matching). */
const CATEGORY_NOTE_PREFIX = 'Category: ';

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
    help: _lt('Onchange: clearing currency clears Amount; setting currency with empty Amount sets 10.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  CurrencyId?: string;

  @Field({
    type: 'monetary',
    // Flat currencyField is accepted at runtime; public FieldOptions overload lags the decorator bag.
    // @ts-expect-error currencyField is validated against this model at Model() load
    currencyField: 'CurrencyId',
    string: _lt('Amount', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('Onchange with Currency: cleared when currency is empty; defaults to 10 when currency is set and amount is empty.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  Amount?: Decimal;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    string: _lt('Active', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('Onchange: toggles Category/Tags domain; inactive widens pickers to include inactive rows.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
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
    help: _lt('Onchange: copies into Progress and narrows Progress selection options.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
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
    help: _lt('Per-company varchar; Category onchange may write "Category: …" here for hand tests.', {
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
    help: _lt('Static domain is active-only; IsActive onchange can widen it. Category onchange writes CompanyNote.', {
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
    help: _lt('ManyToManyRef bound with the Tags widget in Form.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  TagRefIds?: string[];

  @Field<SandboxTag>({
    type: 'ManyToManyRef',
    relation: { targetModel: 'sandbox.SandboxTag' },
    string: _lt('Tag Ref Table', { scope: 'sandbox.model.SandboxRecord.fields' }),
    help: _lt('ManyToManyRef bound with the list/table widget in Form.', {
      scope: 'sandbox.model.SandboxRecord.fields',
    }),
  })
  TagTableRefIds?: string[];

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
    readonly: true,
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
    readonly: true,
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

  // --- kitchen-sink @Onchange demos (value / condition / selection / messages) ---

  /**
   * Narrow or widen Category/Tags pickers from Active; warn when inactive.
   */
  @Onchange<SandboxRecord>('IsActive')
  OnchangeIsActive() {
    const activeOnly = this.IsActive !== false;
    const domain = activeOnly
      ? (['IsActive', '=', true] as const)
      : (['IsActive', 'in', [true, false]] as const);
    const fields = ['CategoryId', 'CategoryRefId', 'Tags', 'TagChips', 'Categories'] as const;
    return {
      condition: fields.map(field => ({ field, condition: [...domain] })),
      // Onchange messages must be plain strings (engine drops non-string message values).
      ...(activeOnly
        ? {}
        : {
            messages: [
              {
                level: 'warn' as const,
                message: 'Inactive record: Category/Tags pickers also include inactive rows.',
              },
            ],
          }),
    };
  }

  /**
   * Mirror State into Progress and filter Progress options by workflow stage.
   */
  @Onchange<SandboxRecord>('State')
  OnchangeState() {
    const state = String(this.State || 'draft');
    this.Progress = state;

    let progressSelection: string[] = [...RECORD_STATE_VALUES];
    let messages: Array<{ level: 'warn'; message: string }> | undefined;
    if (state === 'draft') {
      progressSelection = ['draft', 'open'];
    } else if (state === 'open') {
      progressSelection = ['draft', 'open', 'done'];
    } else if (state === 'done') {
      progressSelection = ['done'];
      messages = [
        {
          level: 'warn',
          message: 'Record marked done; Progress is locked to done.',
        },
      ];
    }

    return {
      selection: [{ field: 'Progress', selection: progressSelection }],
      ...(messages ? { messages } : {}),
    };
  }

  /**
   * Clear Amount when currency is cleared; seed Amount when currency is set and Amount empty.
   */
  @Onchange<SandboxRecord>('CurrencyId')
  OnchangeCurrencyId() {
    const currency = String(this.CurrencyId || '').trim();
    if (!currency) {
      (this as { Amount?: Decimal | null }).Amount = null;
      return;
    }
    if (isEmptyAmount(this.Amount)) {
      this.Amount = new Decimal('10');
    }
  }

  /**
   * Write CompanyNote from Category.Name; clear only when note still matches the prefix.
   */
  @Onchange<SandboxRecord>('CategoryId', { reads: ['CategoryId.Name'] })
  OnchangeCategoryId() {
    const cat = this.CategoryId as { Name?: unknown } | null | undefined;
    const name = cat && cat.Name != null ? String(cat.Name).trim() : '';
    const note = String(this.CompanyNote || '');
    if (!name) {
      if (note.startsWith(CATEGORY_NOTE_PREFIX)) {
        this.CompanyNote = undefined;
      }
      return;
    }
    this.CompanyNote = `${CATEGORY_NOTE_PREFIX}${name}`;
  }
}

function isEmptyAmount(amount: unknown): boolean {
  if (amount == null || amount === '') return true;
  return String(amount).trim() === '';
}
