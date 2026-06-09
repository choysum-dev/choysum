// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import HelloFixture from './fixtures/HelloFixture.vue';

describe('Vue SFC smoke', () => {
  it('mounts a .vue component and renders props', () => {
    const wrapper = mount(HelloFixture, { props: { name: 'Vitest' } });
    expect(wrapper.get('[data-testid="hello"]').text()).toBe('Hello Vitest');
  });
});
