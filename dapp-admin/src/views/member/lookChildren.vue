<template>
    <PageView>
        <a-spin :spinning="loading">
            <a-card title="查看下级">
                <a-breadcrumb style="margin-bottom:16px">
                    <a-breadcrumb-item v-for="(item, idx) in trail" :key="item.user_id || idx">
                        <a v-if="idx < trail.length - 1" @click="jumpTo(idx)">{{ shortAddr(item.address) }}</a>
                        <span v-else>{{ shortAddr(item.address) }}</span>
                    </a-breadcrumb-item>
                </a-breadcrumb>

                <a-descriptions bordered size="small" :column="1" style="margin-bottom:16px">
                    <a-descriptions-item label="当前用户">{{ current.address || '-' }}</a-descriptions-item>
                    <a-descriptions-item label="已支付">{{ current.paid || '0' }}</a-descriptions-item>
                    <a-descriptions-item label="邀请人">
                        <a v-if="inviter" @click="enter(inviter)">{{ inviter.address }}</a>
                        <span v-else>无</span>
                    </a-descriptions-item>
                    <a-descriptions-item label="安置上级">
                        <template v-if="sponsor">
                            <a @click="enter(sponsor)">{{ sponsor.address }}</a>
                            （{{ sideText(sponsor_side) }}）
                        </template>
                        <span v-else>未安置</span>
                    </a-descriptions-item>
                </a-descriptions>

                <a-alert
                    v-if="full"
                    type="info"
                    show-icon
                    message="测试模式：一次展开全部邀请关系与双轨安置。人数上万时不可用于正式环境。"
                    style="margin-bottom:16px"
                />
                <a-alert
                    v-else
                    type="info"
                    show-icon
                    message="正式模式：每次只展示当前用户的直推与左右区。点击节点查看下一层。"
                    style="margin-bottom:16px"
                />

                <div class="org-legend">
                    <span><i class="org-dot current" />当前</span>
                    <span><i class="org-dot invite" />邀请下级</span>
                    <span><i class="org-dot left" />左区</span>
                    <span><i class="org-dot right" />右区</span>
                    <span><i class="org-dot empty" />空位</span>
                </div>

                <a-card type="inner" :title="inviteTitle" style="margin-bottom:16px">
                    <div v-if="inviteRoot" class="org-scroll">
                        <div class="org-tree">
                            <org-branch :node="inviteRoot" @select="enter" />
                        </div>
                    </div>
                    <div v-else>暂无数据</div>
                </a-card>

                <a-card type="inner" :title="placeTitle">
                    <div v-if="placeRoot" class="org-scroll">
                        <div class="org-tree">
                            <org-branch :node="placeRoot" @select="enter" />
                        </div>
                    </div>
                    <div v-else>暂无数据</div>
                </a-card>
            </a-card>
        </a-spin>
    </PageView>
</template>

<script>
import Gai from '../../api/Gai'
import OrgBranch from './OrgBranch.vue'

export default {
    name: 'LookChildrenTree',
    components: { OrgBranch },
    data() {
        return {
            loading: false,
            full: false,
            trail: [],
            current: {},
            inviter: null,
            sponsor: null,
            sponsor_side: '',
            invites: [],
            left: null,
            right: null,
        }
    },
    computed: {
        inviteTitle() {
            const n = this.countInvites(this.invites)
            return this.full ? `邀请关系（全部，共 ${n} 人）` : `邀请关系（直推，共 ${n} 人）`
        },
        placeTitle() {
            return this.full ? '双轨安置（全部）' : '双轨安置（左右区）'
        },
        inviteRoot() {
            if (!this.current || !this.current.address) return null
            return {
                key: 'inv-root-' + this.current.user_id,
                person: this.current,
                label: '当前',
                display: this.shortAddr(this.current.address),
                fullAddress: this.current.address,
                meta: '已支付 ' + (this.current.paid || '0'),
                cls: 'current',
                children: (this.invites || []).map((it) => this.toInviteBranch(it)),
            }
        },
        placeRoot() {
            if (!this.current || !this.current.address) return null
            return {
                key: 'pl-root-' + this.current.user_id,
                person: this.current,
                label: '当前',
                display: this.shortAddr(this.current.address),
                fullAddress: this.current.address,
                meta: '已支付 ' + (this.current.paid || '0'),
                cls: 'current',
                children: [
                    this.toPlaceBranch(this.left, 'L', String(this.current.user_id)),
                    this.toPlaceBranch(this.right, 'R', String(this.current.user_id)),
                ],
            }
        },
    },
    created() {
        this.reloadFromRoute(true)
    },
    activated() {
        this.reloadFromRoute(true)
    },
    methods: {
        reloadFromRoute(resetTrail) {
            const userId = this.$route.query.userId
            const address = this.$route.query.address
            if (resetTrail) this.trail = []
            this.loadUser({ user_id: userId, address }, resetTrail)
        },
        shortAddr(addr) {
            if (!addr) return '-'
            if (addr.length <= 14) return addr
            return addr.slice(0, 6) + '…' + addr.slice(-4)
        },
        sideText(side) {
            if (side === 'L') return '左区'
            if (side === 'R') return '右区'
            return side || '-'
        },
        countInvites(rows) {
            let n = 0
            ;(rows || []).forEach((it) => {
                n += 1
                n += this.countInvites(it.children)
            })
            return n
        },
        toInviteBranch(it) {
            return {
                key: 'inv-' + it.user_id,
                person: it,
                label: '邀请',
                display: this.shortAddr(it.address),
                fullAddress: it.address,
                meta: '已支付 ' + (it.paid || '0'),
                cls: 'invite',
                children: (it.children || []).map((c) => this.toInviteBranch(c)),
            }
        },
        toPlaceBranch(n, side, parentKey) {
            const label = side === 'L' ? '左区' : '右区'
            if (!n) {
                return {
                    key: 'empty-' + parentKey + '-' + side,
                    empty: true,
                    label,
                    display: '空位',
                    fullAddress: '',
                    meta: '',
                    cls: 'empty ' + (side === 'L' ? 'left' : 'right'),
                    children: [],
                }
            }
            const kids = this.full
                ? [
                    this.toPlaceBranch(n.left, 'L', String(n.user_id)),
                    this.toPlaceBranch(n.right, 'R', String(n.user_id)),
                ]
                : []
            return {
                key: 'pl-' + n.user_id + '-' + side,
                person: n,
                label,
                display: this.shortAddr(n.address),
                fullAddress: n.address,
                meta: '子树 ' + (n.amount || '0'),
                cls: side === 'L' ? 'left' : 'right',
                children: kids,
            }
        },
        loadUser(person, resetTrail) {
            if (!person || (!person.user_id && !person.address)) {
                return
            }
            this.loading = true
            Gai.downline({
                user_id: person.user_id,
                userId: person.user_id,
                address: person.address,
            }).then((res) => {
                this.full = !!res.full
                this.current = res.current || {}
                this.inviter = res.inviter || null
                this.sponsor = res.sponsor || null
                this.sponsor_side = res.sponsor_side || ''
                this.invites = res.invites || []
                this.left = res.left || null
                this.right = res.right || null
                const node = {
                    user_id: this.current.user_id,
                    address: this.current.address,
                }
                if (resetTrail) {
                    this.trail = [node]
                } else {
                    const last = this.trail[this.trail.length - 1]
                    if (!last || String(last.user_id) !== String(node.user_id)) {
                        this.trail = this.trail.concat([node])
                    }
                }
            }).finally(() => {
                this.loading = false
            })
        },
        enter(person) {
            if (!person) return
            this.loadUser({ user_id: person.user_id, address: person.address }, false)
        },
        jumpTo(idx) {
            const node = this.trail[idx]
            this.trail = this.trail.slice(0, idx)
            this.loadUser(node, false)
        },
    },
}
</script>

<style lang="less">
.org-legend {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
    margin-bottom: 12px;
    color: #595959;
    font-size: 13px;
    .org-dot {
        display: inline-block;
        width: 10px;
        height: 10px;
        border-radius: 2px;
        margin-right: 6px;
        vertical-align: middle;
        border: 1px solid #d9d9d9;
        &.current { background: #e6f7ff; border-color: #1890ff; }
        &.invite { background: #f6ffed; border-color: #52c41a; }
        &.left { background: #fff7e6; border-color: #fa8c16; }
        &.right { background: #f9f0ff; border-color: #722ed1; }
        &.empty { background: #fafafa; border-style: dashed; }
    }
}
.org-scroll {
    overflow-x: auto;
    padding: 12px 8px 20px;
}
.org-tree {
    display: inline-block;
    min-width: 100%;
    text-align: center;
}
.org-node {
    display: inline-flex;
    flex-direction: column;
    align-items: center;
    vertical-align: top;
}
.org-card {
    min-width: 168px;
    max-width: 220px;
    padding: 10px 12px;
    border-radius: 8px;
    background: #fff;
    border: 2px solid #d9d9d9;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
    cursor: pointer;
    text-align: left;
    position: relative;
    z-index: 1;
}
.org-card:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}
.org-card.current {
    background: #e6f7ff;
    border-color: #1890ff;
}
.org-card.invite {
    background: #f6ffed;
    border-color: #52c41a;
}
.org-card.left {
    background: #fff7e6;
    border-color: #fa8c16;
}
.org-card.right {
    background: #f9f0ff;
    border-color: #722ed1;
}
.org-card.empty {
    background: #fafafa;
    border-style: dashed;
    color: #bfbfbf;
    cursor: default;
    box-shadow: none;
}
.org-tag {
    font-size: 12px;
    color: #8c8c8c;
    margin-bottom: 2px;
}
.org-addr {
    font-family: Menlo, Consolas, monospace;
    font-size: 13px;
    color: #262626;
    word-break: break-all;
}
.org-card.empty .org-addr {
    color: #bfbfbf;
}
.org-meta {
    margin-top: 4px;
    font-size: 12px;
    color: #595959;
}
.org-kids {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 100%;
}
.org-stem {
    width: 2px;
    height: 18px;
    background: #8c8c8c;
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
    padding: 0 16px;
}
.org-joint {
    display: flex;
    width: 100%;
    height: 18px;
    align-items: flex-start;
    .bar {
        flex: 1;
        height: 2px;
        background: #8c8c8c;
        margin-top: 0;
    }
    .drop {
        width: 2px;
        height: 18px;
        background: #8c8c8c;
        flex: none;
    }
}
.org-col.first .org-joint .bar.left,
.org-col.last .org-joint .bar.right,
.org-col.only .org-joint .bar {
    background: transparent;
}
</style>
