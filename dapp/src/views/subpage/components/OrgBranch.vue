<template>
  <div class="org-node">
    <div
      class="org-card"
      :class="node.cls"
      :title="node.fullAddress || ''"
      @click="onSelect"
    >
      <div v-if="node.label" class="org-tag">{{ node.label }}</div>
      <div class="org-addr">{{ node.display }}</div>
      <div v-if="node.meta" class="org-meta">{{ node.meta }}</div>
    </div>
    <div v-if="node.children && node.children.length" class="org-kids">
      <div class="org-stem" />
      <div class="org-row">
        <div
          v-for="(child, idx) in node.children"
          :key="child.key"
          class="org-col"
          :class="{
            first: idx === 0,
            last: idx === node.children.length - 1,
            only: node.children.length === 1
          }"
        >
          <div class="org-joint">
            <i class="bar left" />
            <i class="drop" />
            <i class="bar right" />
          </div>
          <org-branch :node="child" @select="$emit('select', $event)" />
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'OrgBranch',
  props: {
    node: {
      type: Object,
      required: true,
    },
  },
  methods: {
    onSelect() {
      if (this.node.empty || !this.node.person) return
      this.$emit('select', this.node.person)
    },
  },
}
</script>

<style lang="less" scoped>
.org-node {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  vertical-align: top;
}
.org-card {
  min-width: 132px;
  max-width: 180px;
  padding: 8px 10px;
  border-radius: 8px;
  background: #2c333a;
  border: 2px solid #4a5560;
  text-align: left;
  position: relative;
  z-index: 1;
  color: #fff;
}
.org-card.current {
  background: #1d3a55;
  border-color: #3d8fd1;
}
.org-card.invite {
  background: #1e3d2a;
  border-color: #4caf7a;
}
.org-card.left {
  background: #3a2e1d;
  border-color: #d19a3d;
}
.org-card.right {
  background: #332445;
  border-color: #9b6dd4;
}
.org-card.empty {
  background: #262b30;
  border-style: dashed;
  color: #8a9199;
}
.org-tag {
  font-size: 11px;
  color: #9aa3ad;
  margin-bottom: 2px;
}
.org-addr {
  font-family: Menlo, Consolas, monospace;
  font-size: 12px;
  word-break: break-all;
}
.org-card.empty .org-addr {
  color: #8a9199;
}
.org-meta {
  margin-top: 4px;
  font-size: 11px;
  color: #c5ccd3;
}
.org-kids {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
}
.org-stem {
  width: 2px;
  height: 16px;
  background: #6b7680;
}
.org-row {
  display: flex;
  justify-content: center;
  align-items: flex-start;
}
.org-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0 10px;
}
.org-joint {
  display: flex;
  width: 100%;
  height: 16px;
  align-items: flex-start;
  .bar {
    flex: 1;
    height: 2px;
    background: #6b7680;
  }
  .drop {
    width: 2px;
    height: 16px;
    background: #6b7680;
    flex: none;
  }
}
.org-col.first .org-joint .bar.left,
.org-col.last .org-joint .bar.right,
.org-col.only .org-joint .bar {
  background: transparent;
}
.org-tree {
  display: inline-block;
  min-width: 100%;
  text-align: center;
}
</style>

