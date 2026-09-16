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
            <div v-if="node.metaLeft || node.meta" class="org-meta">
                <span v-if="node.metaLeft">{{ node.metaLeft }}</span>
                <span v-if="node.meta">{{ node.meta }}</span>
            </div>
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
