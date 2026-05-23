// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { MetadataStorage } from '@/core/service/api/metadata';
import { buildComputeGraph } from '../compute/graph';
import { buildNeededFields } from './needed';

@Model('test.NeededPartner')
class NeededPartner extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;
}

@Model('test.NeededFieldsModel')
class NeededFieldsModel extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  Status?: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  Code?: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => NeededPartner },
    column: {},
  })
  PartnerId?: NeededPartner;

  @Field({
    type: 'varchar',
    column: {
      size: 128,
      compute: {
        expr: (self: NeededFieldsModel) => `${self.Name || ''}:${self.Code || ''}`,
        deps: ['Name' as any, 'Code' as any],
      },
    },
  })
  Summary?: string;
}

test('buildNeededFields merges onchange reads roots and compute scalar deps', () => {
  const meta = MetadataStorage.instance.getModelMetadata(NeededFieldsModel as any);
  meta.onchangeHandlers = [
    {
      method: 'handleName',
      triggers: ['Name'],
      priority: 100,
      reads: ['Status', 'PartnerId.Name'],
    },
  ];
  meta.computeGraph = buildComputeGraph(meta);

  const result = buildNeededFields(meta, { Name: 'demo' }, ['Name', 'UnknownField']);

  expect(Array.from(result.needed).sort()).toEqual(['Code', 'Name', 'PartnerId', 'Status']);
  expect(result.activeHandlers.map(handler => handler.method)).toEqual(['handleName']);
});

test('buildNeededFields tolerates missing fields/handlers/graph and filters unknown roots', () => {
  const meta = {
    fields: undefined,
    onchangeHandlers: undefined,
    computeGraph: undefined,
  } as any;

  const result = buildNeededFields(meta, {}, ['Unknown', '', 'AlsoUnknown']);

  expect(Array.from(result.needed)).toEqual([]);
  expect(result.activeHandlers).toEqual([]);
});

test('buildNeededFields expands reverse deps transitively and skips missing compute dep entries', () => {
  const meta = {
    fields: new Map<string, any>([
      ['Name', {}],
      ['Status', {}],
      ['Code', {}],
      ['PartnerId', {}],
      ['Summary', {}],
      ['DisplayName', {}],
    ]),
    onchangeHandlers: [
      {
        method: 'handleName',
        triggers: ['Name'],
        reads: ['PartnerId.Name', '.Status'],
      },
      {
        method: 'inactive',
        triggers: ['OtherField'],
        reads: ['Code'],
      },
    ],
    computeGraph: {
      fastReverseDeps: new Map<string, string[]>([
        ['Name', ['Summary']],
        ['Summary', ['DisplayName']],
      ]),
      computeScalarDeps: new Map<string, Set<string>>([['Summary', new Set<string>(['Code'])]]),
    },
  } as any;

  const result = buildNeededFields(meta, { Name: 'n1' }, ['Name']);

  expect(Array.from(result.needed).sort()).toEqual(['Code', 'Name', 'PartnerId', 'Status']);
  expect(result.activeHandlers.map((handler: any) => handler.method)).toEqual(['handleName']);
});

test('buildNeededFields keeps active handler read roots and tolerates sparse compute graph maps', () => {
  const meta = {
    fields: new Map<string, any>([
      ['Name', {}],
      ['PartnerId', {}],
      ['Summary', {}],
      ['Code', {}],
    ]),
    onchangeHandlers: [
      {
        method: 'active',
        triggers: ['Name'],
        reads: ['PartnerId.Name', '', '  ', 'Summary'],
      },
    ],
    computeGraph: {
      fastReverseDeps: new Map<string, string[]>([['Name', ['Summary']]]),
      computeScalarDeps: new Map<string, Set<string>>(),
    },
  } as any;

  const result = buildNeededFields(meta, { Name: 'x' }, ['Name']);

  expect(Array.from(result.needed).sort()).toEqual(['Name', 'PartnerId', 'Summary']);
  expect(result.activeHandlers.map((handler: any) => handler.method)).toEqual(['active']);
});

test('buildNeededFields expands handler trigger set and filters non-string changed entries', () => {
  const meta = {
    fields: new Map<string, any>([
      ['Name', {}],
      ['Code', {}],
      ['Status', {}],
    ]),
    onchangeHandlers: [
      {
        method: 'expand-triggers',
        triggers: ['Name', 'Code'],
        reads: [],
      },
    ],
    computeGraph: {
      fastReverseDeps: new Map<string, string[]>([['Name', []]]),
      computeScalarDeps: new Map<string, Set<string>>(),
    },
  } as any;

  const result = buildNeededFields(meta, {}, [0 as any, '' as any, 'Name']);

  expect(Array.from(result.needed).sort()).toEqual(['Code', 'Name']);
  expect(result.activeHandlers.map((handler: any) => handler.method)).toEqual(['expand-triggers']);
});

test('buildNeededFields tolerates active handler with undefined reads array', () => {
  const meta = {
    fields: new Map<string, any>([
      ['Name', {}],
      ['Status', {}],
    ]),
    onchangeHandlers: [
      {
        method: 'no-reads',
        triggers: ['Name'],
      },
    ],
    computeGraph: undefined,
  } as any;

  const result = buildNeededFields(meta, { Name: 'x' }, ['Name']);

  expect(Array.from(result.needed).sort()).toEqual(['Name']);
  expect(result.activeHandlers.map((handler: any) => handler.method)).toEqual(['no-reads']);
});
