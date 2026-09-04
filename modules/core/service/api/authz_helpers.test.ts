// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  parseConditionEnvelopeFromUnknown,
  parseFieldRuleSpecFromUnknown,
  formatAuthzParseFailureDetail,
  normalizeHitRuleIds,
  replaceConditionExprTokens,
} from './authz_helpers';

function expectParseEnvelopeThrow(value: unknown): void {
  let message = '';
  try {
    parseConditionEnvelopeFromUnknown(value);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(/invalid_record_rule_envelope/.test(message)).toBe(true);
}

function expectParseFieldRuleThrow(value: unknown): void {
  let message = '';
  try {
    parseFieldRuleSpecFromUnknown(value);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(/invalid_field_rule_spec/.test(message)).toBe(true);
}

test('authz helpers parse condition envelope for true/false/expr and reject invalid inputs', () => {
  expectParseEnvelopeThrow(undefined);
  expectParseEnvelopeThrow(null);
  expectParseEnvelopeThrow('not-an-envelope');

  expect(
    parseConditionEnvelopeFromUnknown({
      kind: 'true',
      reason: '  allow  ',
    })
  ).toEqual({
    kind: 'true',
    reason: 'allow',
  });

  expect(
    parseConditionEnvelopeFromUnknown({
      kind: 'false',
      reason: '  denied  ',
    })
  ).toEqual({
    kind: 'false',
    reason: 'denied',
  });

  expect(
    parseConditionEnvelopeFromUnknown({
      kind: 'expr',
      expr: ['OwnerId', '=', '$userId'],
      reason: '  scoped  ',
      hitRuleIds: [' rule_b ', '', 'rule_a', 'rule_a'],
    })
  ).toEqual({
    kind: 'expr',
    expr: ['OwnerId', '=', '$userId'],
    reason: 'scoped',
    hitRuleIds: ['rule_a', 'rule_b'],
  });

  expect(
    parseConditionEnvelopeFromUnknown({
      kind: 'true',
      reason: 'allow',
      hitRuleIds: 'hit_2,hit_1,hit_1',
    })
  ).toEqual({
    kind: 'true',
    reason: 'allow',
    hitRuleIds: ['hit_1', 'hit_2'],
  });

  expect(
    parseConditionEnvelopeFromUnknown({
      kind: 'false',
      reason: 'deny',
      hitRuleIds: ['', '  '],
    })
  ).toEqual({
    kind: 'false',
    reason: 'deny',
  });

  expect(
    parseConditionEnvelopeFromUnknown({
      kind: 'false',
      reason: 'deny_hits',
      hitRuleIds: ['rr_z', 'rr_a'],
    })
  ).toEqual({
    kind: 'false',
    reason: 'deny_hits',
    hitRuleIds: ['rr_a', 'rr_z'],
  });

  expect(
    parseConditionEnvelopeFromUnknown({
      kind: 'expr',
      expr: { And: [['Id', '=', '1']] },
      reason: 'object_expr',
      hitRuleIds: 'only_one',
    })
  ).toEqual({
    kind: 'expr',
    expr: { And: [['Id', '=', '1']] },
    reason: 'object_expr',
    hitRuleIds: ['only_one'],
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: null,
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: {},
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: { Foo: [] },
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: ['OwnerId', '='],
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: ['', '=', '1'],
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: ['  ', '=', '1'],
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: ['Id', '', '1'],
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: ['Id', '  ', '1'],
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: ['Id', '=', undefined],
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: ['Id', '=', () => 'x'],
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: ['Id', '=', Symbol('x')],
  });

  expect(
    parseConditionEnvelopeFromUnknown({
      kind: 'expr',
      expr: ['DeletedAt', '=', null],
    })
  ).toEqual({
    kind: 'expr',
    expr: ['DeletedAt', '=', null],
    reason: undefined,
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: 'not-a-condition',
    reason: 'bad_expr',
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: { And: 'not-array' },
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: { Or: null },
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: { And: [] },
  });

  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: { Or: [] },
  });

  let deep: unknown = ['Id', '=', '1'];
  for (let i = 0; i < 33; i += 1) {
    deep = { And: [deep] };
  }
  expectParseEnvelopeThrow({
    kind: 'expr',
    expr: deep,
  });

  expectParseEnvelopeThrow({
    kind: 'unknown',
    reason: 'x',
  });

  expect(
    parseConditionEnvelopeFromUnknown({
      kind: 'expr',
      expr: ['OwnerId', '=', '1'],
      reason: 'no_hits',
    })
  ).toEqual({
    kind: 'expr',
    expr: ['OwnerId', '=', '1'],
    reason: 'no_hits',
  });
});

test('authz helpers recursively replace condition tokens in nested expressions', () => {
  const out = replaceConditionExprTokens(
    {
      And: [
        ['OwnerId', '=', '$userId'],
        {
          Or: [
            ['CompanyId', '=', '$companyId'],
            ['CompanyId', 'in', '$companyIds'],
            ['Marker', '=', 'keep'],
          ],
        },
      ],
    } as any,
    {
      userId: 'usr_1',
      companyId: 'cmp_1',
      companyIds: ['cmp_1', 'cmp_2'],
      strictUnknownToken: true,
    }
  );

  expect(out).toEqual({
    And: [
      ['OwnerId', '=', 'usr_1'],
      {
        Or: [
          ['CompanyId', '=', 'cmp_1'],
          ['CompanyId', 'in', ['cmp_1', 'cmp_2']],
          ['Marker', '=', 'keep'],
        ],
      },
    ],
  });
});

test('authz helpers strict token mode rejects unknown and missing token values', () => {
  let message = '';
  try {
    replaceConditionExprTokens(['OwnerId', '=', '$tenantId'] as any, {
      userId: 'usr_1',
      companyId: 'cmp_1',
      companyIds: ['cmp_1'],
      strictUnknownToken: true,
    });
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('unknown condition token: $tenantId')).toBe(true);

  message = '';
  try {
    replaceConditionExprTokens(['OwnerId', '=', '$userId'] as any, {
      strictUnknownToken: true,
    });
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('missing token value: $userId')).toBe(true);
});

test('authz helpers non-strict mode keeps unknown tokens unchanged', () => {
  const out = replaceConditionExprTokens(['OwnerId', '=', '$tenantId'] as any, {
    userId: 'usr_1',
    companyId: 'cmp_1',
    companyIds: ['cmp_1'],
  });

  expect(out).toEqual(['OwnerId', '=', '$tenantId']);
});

test('authz helpers parse field rule spec and reject invalid payloads', () => {
  expectParseFieldRuleThrow(undefined);
  expectParseFieldRuleThrow(null);
  expectParseFieldRuleThrow('not-a-spec');

  expect(
    parseFieldRuleSpecFromUnknown({
      denyReadFields: [' Name ', 'Name', 'Id'],
      denyWriteFields: [' Amount ', 'Amount', 'Locked'],
      reason: '  from_auth  ',
      hitRuleIds: ['fr_2', 'fr_1', 'fr_1'],
    })
  ).toEqual({
    denyReadFields: ['Name', 'Id'],
    denyWriteFields: ['Amount', 'Locked'],
    reason: 'from_auth',
    hitRuleIds: ['fr_1', 'fr_2'],
  });

  expect(
    parseFieldRuleSpecFromUnknown({
      reason: '   ',
    })
  ).toEqual({
    denyReadFields: [],
    denyWriteFields: [],
    reason: undefined,
  });

  expect(
    parseFieldRuleSpecFromUnknown({
      denyReadFields: null,
      denyWriteFields: null,
    })
  ).toEqual({
    denyReadFields: [],
    denyWriteFields: [],
    reason: undefined,
  });

  expectParseFieldRuleThrow({
    denyReadFields: 'not-array',
    denyWriteFields: [],
  });
  expectParseFieldRuleThrow({
    denyReadFields: [],
    denyWriteFields: 1,
  });
  expectParseFieldRuleThrow({
    denyReadFields: [null],
    denyWriteFields: [],
  });
  expectParseFieldRuleThrow({
    denyReadFields: [1],
    denyWriteFields: [],
  });
  expectParseFieldRuleThrow({
    denyReadFields: [''],
    denyWriteFields: [],
  });
  expectParseFieldRuleThrow({
    denyReadFields: ['  '],
    denyWriteFields: [],
  });
  expectParseFieldRuleThrow({
    denyReadFields: [],
    denyWriteFields: [false],
  });

  expect(
    parseFieldRuleSpecFromUnknown({
      denyReadFields: [],
      denyWriteFields: [],
      hitRuleIds: ' fr_b , ,fr_a ',
    })
  ).toEqual({
    denyReadFields: [],
    denyWriteFields: [],
    reason: undefined,
    hitRuleIds: ['fr_a', 'fr_b'],
  });
});

test('authz helpers formatAuthzParseFailureDetail covers Error and non-Error', () => {
  expect(formatAuthzParseFailureDetail(new Error('invalid_field_rule_spec'))).toBe('invalid_field_rule_spec');
  expect(formatAuthzParseFailureDetail('raw-detail')).toBe('raw-detail');
  expect(formatAuthzParseFailureDetail(42)).toBe('42');
});

test('authz helpers normalizeHitRuleIds covers array string and non-list inputs', () => {
  expect(normalizeHitRuleIds([' b ', '', 'a', 'a', null, undefined, 0])).toEqual(['0', 'a', 'b']);
  expect(normalizeHitRuleIds([])).toBe(undefined);
  expect(normalizeHitRuleIds(['', '  '])).toBe(undefined);
  expect(normalizeHitRuleIds(' z ,y, y ,')).toEqual(['y', 'z']);
  expect(normalizeHitRuleIds('')).toBe(undefined);
  expect(normalizeHitRuleIds(', ,')).toBe(undefined);
  expect(normalizeHitRuleIds(undefined)).toBe(undefined);
  expect(normalizeHitRuleIds(12 as any)).toBe(undefined);
});

// Local dial (ORM / document owner) and gRPC Value results both feed these parsers;
// invalid shapes must fail the same way (no wash-to-allow / wash-to-false at D5).
test('authz D5 parsers: local and gRPC consumers share identical fail-hard errors', () => {
  const badEnvelopes = [null, undefined, 'not-envelope', 1, [], { kind: 'maybe' }, { kind: 'expr' }];
  for (const value of badEnvelopes) {
    expect(() => parseConditionEnvelopeFromUnknown(value)).toThrow(/invalid_record_rule_envelope/);
  }

  const badFieldSpecs = [null, undefined, 'not-spec', 1, []];
  for (const value of badFieldSpecs) {
    expect(() => parseFieldRuleSpecFromUnknown(value)).toThrow(/invalid_field_rule_spec/);
  }
});
