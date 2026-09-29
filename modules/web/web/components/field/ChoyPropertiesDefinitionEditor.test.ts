// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick } from 'vue';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyPropertiesDefinitionEditor from './ChoyPropertiesDefinitionEditor.vue';

function installChoyButtonStub() {
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    props: {
      disabled: Boolean,
      loading: Boolean,
      type: String,
      size: String,
      variant: String,
    },
    emits: ['click'],
    inheritAttrs: false,
    setup(props: any, { emit, slots, attrs }: any) {
      return () =>
        h(
          'button',
          {
            ...attrs,
            type: 'button',
            disabled: props.disabled || undefined,
            'data-disabled': props.disabled ? '1' : '0',
            onClick: (e: Event) => emit('click', e),
          },
          slots.default?.()
        );
    },
  } as any);
}

function setInputValue(root: Element, selector: string, value: string) {
  const el = root.querySelector(selector) as HTMLInputElement | HTMLTextAreaElement | null;
  expect(el).toBeTruthy();
  el!.value = value;
  el!.dispatchEvent(new Event('input', { bubbles: true }));
}

describe('ChoyPropertiesDefinitionEditor', () => {
  beforeEach(() => {
    installChoyButtonStub();
  });

  afterEach(() => {
    restoreSfc(ChoyButton as any);
  });

  test('Search loads drafts and Save updates by id', async () => {
    const Search = fnRecorder(async () => [
      {
        Id: 'def-1',
        Definition: [{ name: 'tax_id', type: 'char', string: 'Tax' }],
      },
    ]);
    const UpdateById = fnRecorder(async () => ({ Id: 'def-1' }));
    const Create = fnRecorder(async () => ({ Id: 'new' }));
    const store = { Search, UpdateById, Create } as any;

    const mounted = mountApp(ChoyPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        store,
      },
    });
    try {
      await flushPromises();
      expect(Search.calls.length).toBeGreaterThan(0);
      const andCond = (Search.calls[0]![0] as any).And;
      expect(andCond).toEqual([
        ['TargetModel', '=', 'Partner'],
        ['PropertiesField', '=', 'PartnerProperties'],
        ['ContainerModel', '=', null],
        ['ContainerId', '=', null],
      ]);
      expect(mounted.q('[data-testid="o-properties-definition-name"]')).toBeTruthy();

      mounted.click('[data-testid="o-properties-definition-save"]');
      await flushPromises();
      expect(UpdateById.calls.length).toBe(1);
      expect(UpdateById.calls[0]![0]).toBe('def-1');
      expect((UpdateById.calls[0]![1] as any).Definition[0].name).toBe('tax_id');
      expect(Create.calls.length).toBe(0);
    } finally {
      mounted.unmount();
    }
  });

  test('empty Search then Add+Save creates a parent-scoped definition', async () => {
    const Search = fnRecorder(async () => []);
    const Create = fnRecorder(async (vals: any) => ({ Id: 'new-1', ...vals }));
    const UpdateById = fnRecorder(async () => ({}));
    const store = { Search, Create, UpdateById } as any;

    const mounted = mountApp(ChoyPropertiesDefinitionEditor as any, {
      props: {
        application: 'project',
        targetModel: 'Task',
        propertiesField: 'TaskProperties',
        containerModel: 'Project',
        containerId: 'proj-9',
        store,
      },
    });
    try {
      await flushPromises();
      expect(mounted.q('[data-testid="o-properties-definition-empty"]')).toBeTruthy();

      mounted.click('[data-testid="o-properties-definition-add"]');
      await nextTick();
      setInputValue(mounted.el, '[data-testid="o-properties-definition-name"]', 'prio');
      mounted.click('[data-testid="o-properties-definition-save"]');
      await flushPromises();

      expect(Create.calls.length).toBe(1);
      const payload = Create.calls[0]![0] as any;
      expect(payload.TargetModel).toBe('Task');
      expect(payload.PropertiesField).toBe('TaskProperties');
      expect(payload.ContainerModel).toBe('Project');
      expect(payload.ContainerId).toBe('proj-9');
      expect(payload.Definition[0].name).toBe('prio');
      expect(UpdateById.calls.length).toBe(0);
    } finally {
      mounted.unmount();
    }
  });

  test('readonly disables Save and Add via DOM', async () => {
    const Search = fnRecorder(async () => [
      { Id: 'r1', Definition: [{ name: 'a', type: 'char' }] },
    ]);
    const Create = fnRecorder(async () => ({}));
    const UpdateById = fnRecorder(async () => ({}));
    const store = { Search, Create, UpdateById } as any;

    const mounted = mountApp(ChoyPropertiesDefinitionEditor as any, {
      props: {
        application: 'partner',
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
        readonly: true,
        store,
      },
    });
    try {
      await flushPromises();
      const save = mounted.q('[data-testid="o-properties-definition-save"]') as HTMLButtonElement | null;
      const add = mounted.q('[data-testid="o-properties-definition-add"]') as HTMLButtonElement | null;
      expect(save).toBeTruthy();
      expect(add).toBeTruthy();
      expect(save!.getAttribute('data-disabled')).toBe('1');
      expect(add!.getAttribute('data-disabled')).toBe('1');
      expect(Create.calls.length).toBe(0);
      expect(UpdateById.calls.length).toBe(0);
    } finally {
      mounted.unmount();
    }
  });
});
