// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

type DepSubscriber = {
  update: () => void;
};

/**
 * Dep stores subscriptions for a reactive field dependency.
 */
export class Dep {
  private subscribers = new Set<DepSubscriber>();

  /**
   * Holds the watcher currently collecting dependencies.
   */
  static target: DepSubscriber | null = null;

  /**
   * Registers the active watcher as a subscriber.
   */
  depend() {
    if (Dep.target) {
      this.subscribers.add(Dep.target);
    }
  }

  /**
   * Notifies all subscribers that the dependency changed.
   */
  notify() {
    this.subscribers.forEach(watcher => watcher.update());
  }
}
