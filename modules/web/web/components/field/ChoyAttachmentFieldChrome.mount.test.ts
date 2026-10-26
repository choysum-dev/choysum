// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyAttachmentFieldChrome from './ChoyAttachmentFieldChrome.vue';

function installButtonStub() {
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    props: {
      class: { type: [String, Object, Array], default: undefined },
      disabled: Boolean,
      size: String,
      variant: String,
      type: String,
      as: String,
    },
    setup(p: any, { attrs, slots }: any) {
      return () =>
        h(
          'button',
          { type: 'button', class: p.class, disabled: p.disabled, ...attrs },
          slots.default?.()
        );
    },
  });
}

const baseLabels = {
  uploadLabel: 'Upload',
  dropLabel: 'Drop here',
  replaceLabel: 'Replace',
  removeLabel: 'Remove',
  openLabel: 'Download',
};

describe('ChoyAttachmentFieldChrome', () => {
  afterEach(() => {
    restoreSfc(ChoyButton as any);
  });

  test('file presentation: empty edit shows upload and accepts file change', async () => {
    installButtonStub();
    const files: File[] = [];
    const m = mountApp(ChoyAttachmentFieldChrome as any, {
      props: {
        ...baseLabels,
        mode: 'edit',
        hasValue: false,
        presentation: 'file',
        accept: 'application/pdf',
        onFile: (f: File) => files.push(f),
      },
    });
    await flushPromises();
    expect(m.q('.choy-attachment-field')).toBeTruthy();
    const input = m.q('input.sr-only') as HTMLInputElement | null;
    expect(input).toBeTruthy();
    const file = new File([new Uint8Array([1, 2, 3])], 'a.pdf', { type: 'application/pdf' });
    Object.defineProperty(input, 'files', {
      configurable: true,
      value: { 0: file, length: 1, item: (i: number) => (i === 0 ? file : null) },
    });
    input!.dispatchEvent(new Event('change', { bubbles: true }));
    await flushPromises();
    expect(files.length).toBe(1);
    expect(files[0]?.name).toBe('a.pdf');
    m.unmount();
  });

  test('image presentation: valued edit renders avatar chrome and remove', async () => {
    installButtonStub();
    const m = mountApp(ChoyAttachmentFieldChrome as any, {
      props: {
        ...baseLabels,
        mode: 'edit',
        hasValue: true,
        presentation: 'image',
        openHref: '/_document/bindings/b1/content',
        downloadFileName: 'photo.png',
        width: 96,
        height: 96,
        fit: 'cover',
        radius: 'full',
      },
      slots: {
        media: () => h('img', { alt: 'preview', src: 'data:image/png;base64,aa' }),
      },
    });
    await flushPromises();
    expect(m.q('.choy-image-avatar')).toBeTruthy();
    const ss = m.setupState() as any;
    expect(!!(ss.useImagePresentation?.value ?? ss.useImagePresentation)).toBe(true);
    expect(String(ss.effectiveFit?.value ?? ss.effectiveFit)).toBe('cover');
    expect(typeof ss.onPreviewClick).toBe('function');
    ss.onPreviewClick?.();
    // Empty href short-circuit (avoid window.open in QuickJS host).
    let prevented = false;
    const m2 = mountApp(ChoyAttachmentFieldChrome as any, {
      props: {
        ...baseLabels,
        mode: 'edit',
        hasValue: true,
        presentation: 'image',
        openHref: '',
        width: 64,
        height: 64,
      },
    });
    await flushPromises();
    const ss2 = m2.setupState() as any;
    await ss2.onDownloadClick?.({ preventDefault: () => { prevented = true; } });
    expect(prevented).toBe(true);
    m2.unmount();
    m.unmount();
  });

  test('compact valued row and disabled download short-circuit', async () => {
    installButtonStub();
    const m = mountApp(ChoyAttachmentFieldChrome as any, {
      props: {
        ...baseLabels,
        mode: 'edit',
        hasValue: true,
        presentation: 'file',
        compact: true,
        openHref: '/_document/bindings/b1/content',
        downloadFileName: 'doc.bin',
        disabled: true,
        tooltip: 'application/octet-stream · 1 KB',
        title: 'doc.bin',
      },
    });
    await flushPromises();
    expect(m.q('.choy-attachment-field')).toBeTruthy();
    const ss = m.setupState() as any;
    expect(ss.actionsDisabled).toBe(true);
    let prevented = false;
    await ss.onDownloadClick?.({ preventDefault: () => { prevented = true; } });
    expect(prevented).toBe(true);
    m.unmount();
  });

  test('busy valued state marks uploading and ignores file input while disabled', async () => {
    installButtonStub();
    const files: File[] = [];
    const m = mountApp(ChoyAttachmentFieldChrome as any, {
      props: {
        ...baseLabels,
        mode: 'edit',
        hasValue: true,
        presentation: 'file',
        busy: true,
        openHref: '/files/x',
        onFile: (f: File) => files.push(f),
      },
    });
    await flushPromises();
    const ss = m.setupState() as any;
    expect(ss.attachmentState).toBe('uploading');
    const input = m.q('input.sr-only') as HTMLInputElement | null;
    if (input) {
      const file = new File([new Uint8Array([9])], 'x.bin');
      Object.defineProperty(input, 'files', {
        configurable: true,
        value: { 0: file, length: 1, item: () => file },
      });
      input.dispatchEvent(new Event('change', { bubbles: true }));
      await flushPromises();
    }
    expect(files.length).toBe(0);
    // Drag/drop short-circuit while busy.
    ss.onDragOver?.({ dataTransfer: { dropEffect: '' } });
    ss.onDrop?.({ dataTransfer: { files: [new File([new Uint8Array([1])], 'd.bin')] } });
    expect(files.length).toBe(0);
    m.unmount();
  });
});
