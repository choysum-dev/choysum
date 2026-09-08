// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Relation typeahead fields need useField, permission, and store mocks at setup;
// full mount needs store/composable stubs at setup; import smoke only for this knife.

test('relation typeahead NameSearch smoke: related field SFCs export components', async () => {
  const manyToOne = await import('./OManyToOneField.vue');
  const manyToOneRef = await import('./OManyToOneRefField.vue');
  const tags = await import('./OManyToManyTagsField.vue');
  const refTags = await import('./OManyToManyRefTagsField.vue');
  for (const mod of [manyToOne, manyToOneRef, tags, refTags]) {
    expect(mod.default).toBeTruthy();
    const name = (mod.default as { name?: string; __name?: string }).name
      || (mod.default as { name?: string; __name?: string }).__name;
    expect(name).toBeTruthy();
  }
});
