<template>
    <PageView>
        <a-card title="日封顶档位">
            <div class="toolbar">
                <span class="hint">每一档是一个金额区间：填写该档上限后，列表按上限金额从小到大排列。最后一档没有上限。保存后立刻影响结算和商城展示。</span>
                <div>
                    <a-button @click="addRow">添加档位</a-button>
                    <a-button type="primary" :loading="saving" style="margin-left:8px;" @click="save">保存</a-button>
                </div>
            </div>
            <a-table
                :loading="loading"
                :columns="columns"
                :dataSource="rows"
                :pagination="false"
                bordered
                :scroll="{ x: true }"
                rowKey="key"
            />
            <div class="preview">
                <span>试算合计</span>
                <a-input-number v-model="sample" :min="0" style="width:160px;margin:0 8px;" />
                <span>USDT → {{ sampleRange }}，日封顶 {{ sampleCap }} USDT</span>
            </div>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'

const trimNum = (v) => {
    const s = String(v == null ? '' : v).trim()
    if (!s) return ''
    if (!/^-?\d+(\.\d+)?$/.test(s)) return s
    return s.replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '')
}

const prevMax = (rows, index) => {
    for (let i = index - 1; i >= 0; i--) {
        const max = trimNum(rows[i] && rows[i].max_amount)
        if (max) return max
    }
    return '0'
}

const rangeForAmount = (amount, tiers) => {
    const a = Number(amount)
    if (!Number.isFinite(a) || a <= 0) return '—'
    const list = Array.isArray(tiers) ? tiers : []
    let from = '0'
    for (let i = 0; i < list.length; i++) {
        const max = trimNum(list[i] && list[i].max_amount)
        if (!max) return `${from} 及以上`
        if (a < Number(max)) return `${from} ≤ 金额 < ${max}`
        from = max
    }
    return list.length ? `${from} 及以上` : '—'
}

const capForTiers = (amount, tiers) => {
    const a = Number(amount)
    if (!Number.isFinite(a) || a <= 0) return '0'
    const list = Array.isArray(tiers) && tiers.length ? tiers : []
    for (let i = 0; i < list.length; i++) {
        const max = String(list[i].max_amount || '').trim()
        const cap = String(list[i].daily_cap || '0')
        if (!max) return cap
        if (a < Number(max)) return cap
    }
    return list.length ? String(list[list.length - 1].daily_cap || '0') : '0'
}

export default {
    name: 'dailyCap',
    data() {
        return {
            loading: false,
            saving: false,
            sample: 3000,
            rows: [],
            columns: [
                {
                    title: '序号',
                    width: 70,
                    customRender: (v, row, index) => index + 1,
                },
                {
                    title: '金额区间',
                    customRender: (v, row, index) => {
                        const from = prevMax(this.rows, index)
                        if (this.isOpenEnded(index)) {
                            return <span>{from} 及以上</span>
                        }
                        return (
                            <div style="display:flex;align-items:center;white-space:nowrap;">
                                <span style="margin-right:8px;">{from} ≤ 金额 &lt;</span>
                                <a-input
                                    value={row.max_amount}
                                    placeholder="填写上限"
                                    style="width:140px"
                                    onInput={(e) => this.setField(index, 'max_amount', e.target.value)}
                                    onBlur={() => this.sortRows()}
                                />
                            </div>
                        )
                    },
                },
                {
                    title: '日封顶',
                    dataIndex: 'daily_cap',
                    width: 180,
                    customRender: (v, row, index) => {
                        return <a-input value={row.daily_cap} placeholder="日封顶" onInput={(e) => this.setField(index, 'daily_cap', e.target.value)} />
                    },
                },
                {
                    title: '操作',
                    key: 'action',
                    width: 90,
                    customRender: (v, row, index) => {
                        return <a-button type="danger" ghost onClick={() => this.removeRow(index)}>删除</a-button>
                    },
                },
            ],
        }
    },
    computed: {
        sampleCap() {
            return capForTiers(this.sample, this.rows)
        },
        sampleRange() {
            return rangeForAmount(this.sample, this.rows)
        },
    },
    activated() {
        this.getList()
    },
    created() {
        this.getList()
    },
    methods: {
        isOpenEnded(index) {
            return index === this.rows.length - 1
        },
        setField(index, field, value) {
            if (!this.rows[index]) return
            this.$set(this.rows[index], field, value)
        },
        rowKey() {
            return `cap-${Date.now()}-${Math.random().toString(16).slice(2)}`
        },
        mapRows(tiers) {
            const rows = (tiers || []).map((row, i) => ({
                key: `cap-${i}-${row.max_amount}-${row.daily_cap}`,
                max_amount: row.max_amount == null ? '' : String(row.max_amount),
                daily_cap: row.daily_cap == null ? '' : String(row.daily_cap),
            }))
            return this.sortedRows(rows)
        },
        amountValue(row) {
            const raw = trimNum(row && row.max_amount)
            if (!raw) return null
            const n = Number(raw)
            return Number.isFinite(n) ? n : null
        },
        sortedRows(list) {
            const rows = Array.isArray(list) ? list.slice() : []
            if (rows.length <= 1) return rows
            const open = rows[rows.length - 1]
            const bounded = rows.slice(0, -1)
            bounded.sort((a, b) => {
                const av = this.amountValue(a)
                const bv = this.amountValue(b)
                if (av == null && bv == null) return 0
                if (av == null) return 1
                if (bv == null) return -1
                return av - bv
            })
            return bounded.concat(open)
        },
        sortRows() {
            this.rows = this.sortedRows(this.rows)
        },
        getList() {
            this.loading = true
            Gai.daily_cap_tiers().then((res) => {
                this.rows = this.mapRows(res && res.tiers)
                this.loading = false
            }).catch(() => {
                this.loading = false
            })
        },
        addRow() {
            if (this.rows.length >= 20) {
                this.$message.warning('最多 20 档')
                return
            }
            const row = { key: this.rowKey(), max_amount: '', daily_cap: '' }
            if (this.rows.length) {
                this.rows.splice(this.rows.length - 1, 0, row)
            } else {
                this.rows.push(row)
            }
            this.sortRows()
        },
        removeRow(index) {
            if (this.rows.length <= 1) {
                this.$message.warning('至少保留一档')
                return
            }
            this.rows.splice(index, 1)
            this.sortRows()
        },
        save() {
            this.sortRows()
            const tiers = this.rows.map((row, i) => ({
                max_amount: this.isOpenEnded(i) ? '' : String(row.max_amount || '').trim(),
                daily_cap: String(row.daily_cap || '').trim(),
            }))
            this.saving = true
            Gai.daily_cap_tiers_update({ tiers }).then((res) => {
                this.rows = this.mapRows(res && res.tiers)
                this.$message.success('已保存')
            }).finally(() => {
                this.saving = false
            })
        },
    },
}
</script>

<style scoped lang="less">
.toolbar {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 16px;
    margin-bottom: 16px;
}
.hint {
    color: rgba(0, 0, 0, 0.45);
    font-size: 13px;
    line-height: 1.6;
    flex: 1;
}
.preview {
    margin-top: 16px;
    color: rgba(0, 0, 0, 0.65);
}
</style>
