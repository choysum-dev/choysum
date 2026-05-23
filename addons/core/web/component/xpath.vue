<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script lang="ts">
import { defineComponent, PropType } from 'vue';

type PositionType = 'before' | 'after' | 'inside' | 'replace' | 'attribute';
type XPathExpr = string;

type EmptyProps = {
  expr?: never;
  position?: never;
  attrName?: never;
  attrValue?: never;
};

type StandardXPathProps = {
  expr: XPathExpr;
  position: Exclude<PositionType, 'attribute'>;
  attrName?: never;
  attrValue?: never;
};

type AttributeXPathProps = {
  expr: XPathExpr;
  position: 'attribute';
  attrName: string;
  attrValue: string;
};

type XPathProps = EmptyProps | StandardXPathProps | AttributeXPathProps;

export default defineComponent<XPathProps>({
  props: {
    expr: {
      type: String as PropType<XPathExpr>,
      required: false,
    },
    position: {
      type: String as PropType<PositionType>,
      required: false,
    },
    attrName: {
      type: String,
      required: false,
    },
    attrValue: {
      type: String,
      required: false,
    },
  },
});
</script>
