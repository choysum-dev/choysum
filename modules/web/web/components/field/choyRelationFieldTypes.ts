// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type ChoyManyToManyWidget = 'tags' | 'list' | 'tree';

export type ChoyManyToManyTreeNode = {
  id: string;
  label: string;
  children?: ChoyManyToManyTreeNode[];
};

export type ChoyOneToManyWidget = 'list' | 'kanban';
