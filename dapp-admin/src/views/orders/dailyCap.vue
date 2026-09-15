<template>
    <PageView>
        <a-card title="日封顶档位">
            <div class="toolbar">
                <span class="hint">金额小于「合计上限」时套该行日封顶；最后一行上限留空，表示以上全部。保存后立刻影响结算与商城购物车展示。</span>
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
                <span>USDT → 日封顶 {{ sampleCap }} USDT</span>
            </div>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'

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
                    title: '合计上限（不含）',
                    dataIndex: 'max_amount',
                    customRender: (v, row, index) => {
                        return <a-input value={row.max_amount} placeholder="最后一档留空" onInput={(e) => this.setField(index, 'max_amount', e.target.value)} />
                    },
                },
                {
                    title: '日封顶',
                    dataIndex: 'daily_cap',
                    customRender: (v, row, index) => {
                        return <a-input value={row.daily_cap} placeholder="日封顶" onInput={(e) => this.setField(index, 'daily_cap', e.target.value)} />
                    },
                },
                {
                    title: '规则预览',
                    customRender: (v, row) => {
                        const cap = row.daily_cap || '0'
                        if (!String(row.max_amount || '').trim()) {
                            return <span>合计达到以上档位 → {cap}</span>
                        }
                        return <span>合计 &lt; {row.max_amount} → {cap}</span>
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
    },
    activated() {
        this.getList()
    },
    created() {
        this.getList()
    },
    methods: {
        setField(index, field, value) {
            if (!this.rows[index]) return
            this.$set(this.rows[index], field, value)
        },
        rowKey(i) {
            return `cap-${i}-${Date.now()}`
        },
        mapRows(tiers) {
            return (tiers || []).map((row, i) => ({
                key: `cap-${i}-${row.max_amount}-${row.daily_cap}`,
                max_amount: row.max_amount == null ? '' : String(row.max_amount),
                daily_cap: row.daily_cap == null ? '' : String(row.daily_cap),
            }))
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
            const row = { key: this.rowKey(this.rows.length), max_amount: '', daily_cap: '' }
            const last = this.rows[this.rows.length - 1]
            if (last && !String(last.max_amount || '').trim()) {
                this.rows.splice(this.rows.length - 1, 0, row)
            } else {
                this.rows.push(row)
            }
        },
        removeRow(index) {
            if (this.rows.length <= 1) {
                this.$message.warning('至少保留一档')
                return
            }
            this.rows.splice(index, 1)
        },
        save() {
            const tiers = this.rows.map((row) => ({
                max_amount: String(row.max_amount || '').trim(),
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
