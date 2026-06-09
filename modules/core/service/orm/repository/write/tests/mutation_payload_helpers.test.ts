// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyRepositoryMutationDefaultValues,
  assertRepositoryMutationPayloadsAllowed,
  encodeRepositoryMutationPayloads,
  validateRepositoryMutationPayload,
} from '../mutation_payload_helpers';

test('repository mutation payload helpers assert field-rule access for each payload', async () => {
  const calls: Array<Record<string, any>> = [];
  await assertRepositoryMutationPayloadsAllowed(
    {
      async assertFieldRuleWriteAllowed(payload) {
        calls.push({ method: 'fieldRule', payload });
      },
    },
    [{ Name: 'first' } as any, { Name: 'second' } as any]
  );

  expect(calls).toEqual([
    { method: 'fieldRule', payload: { Name: 'first' } },
    { method: 'fieldRule', payload: { Name: 'second' } },
  ]);
});

test('repository mutation payload helpers apply default values across payloads', () => {
  const calls: Array<Record<string, any>> = [];
  const payloads = applyRepositoryMutationDefaultValues(
    {
      applyDefaultMutationValues(payload) {
        calls.push({ method: 'defaultValues', payload });
        return { ...payload, CompanyId: 'company_a' } as any;
      },
    },
    [{ Name: 'first' } as any, { Name: 'second' } as any]
  );

  expect(payloads).toEqual([
    { Name: 'first', CompanyId: 'company_a' },
    { Name: 'second', CompanyId: 'company_a' },
  ]);
  expect(calls).toEqual([
    { method: 'defaultValues', payload: { Name: 'first' } },
    { method: 'defaultValues', payload: { Name: 'second' } },
  ]);
});

test('repository mutation payload helpers validate against each provided context', async () => {
  const calls: Array<Record<string, any>> = [];
  await validateRepositoryMutationPayload(
    {
      async validateFields(input, mode, current) {
        calls.push({ method: 'validate', input, mode, current });
      },
    },
    { Name: 'demo' } as any,
    'update',
    [{ Id: 'row_1' }, undefined, { Id: 'row_2' }]
  );

  expect(calls).toEqual([
    { method: 'validate', input: { Name: 'demo' }, mode: 'update', current: { Id: 'row_1' } },
    { method: 'validate', input: { Name: 'demo' }, mode: 'update', current: undefined },
    { method: 'validate', input: { Name: 'demo' }, mode: 'update', current: { Id: 'row_2' } },
  ]);
});

test('repository mutation payload helpers default validation context to undefined when omitted', async () => {
  const calls: Array<Record<string, any>> = [];
  await validateRepositoryMutationPayload(
    {
      async validateFields(input, mode, current) {
        calls.push({ method: 'validate', input, mode, current });
      },
    },
    { Name: 'demo' } as any,
    'create'
  );

  expect(calls).toEqual([{ method: 'validate', input: { Name: 'demo' }, mode: 'create', current: undefined }]);
});

test('repository mutation payload helpers encode payloads in order', () => {
  const calls: Array<Record<string, any>> = [];
  const payloads = encodeRepositoryMutationPayloads(
    {
      encodeForDb(input) {
        calls.push({ method: 'encode', input });
        return { ...input, Encoded: true } as any;
      },
    },
    [{ Name: 'first' } as any, { Name: 'second' } as any]
  );

  expect(payloads).toEqual([
    { Name: 'first', Encoded: true },
    { Name: 'second', Encoded: true },
  ]);
  expect(calls).toEqual([
    { method: 'encode', input: { Name: 'first' } },
    { method: 'encode', input: { Name: 'second' } },
  ]);
});

test('repository mutation payload helpers treat undefined payload list as empty for guard/default/encode steps', async () => {
  const guardCalls: Array<Record<string, any>> = [];
  await assertRepositoryMutationPayloadsAllowed(
    {
      async assertFieldRuleWriteAllowed(payload) {
        guardCalls.push({ method: 'fieldRule', payload });
      },
    },
    undefined as any
  );
  expect(guardCalls).toEqual([]);

  const defaultsCalls: Array<Record<string, any>> = [];
  const defaultsOut = applyRepositoryMutationDefaultValues(
    {
      applyDefaultMutationValues(payload) {
        defaultsCalls.push({ method: 'defaultValues', payload });
        return payload;
      },
    },
    undefined as any
  );
  expect(defaultsOut).toEqual([]);
  expect(defaultsCalls).toEqual([]);

  const encodeCalls: Array<Record<string, any>> = [];
  const encoded = encodeRepositoryMutationPayloads(
    {
      encodeForDb(input) {
        encodeCalls.push({ method: 'encode', input });
        return input;
      },
    },
    undefined as any
  );
  expect(encoded).toEqual([]);
  expect(encodeCalls).toEqual([]);
});
