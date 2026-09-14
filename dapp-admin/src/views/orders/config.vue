<template>
    <PageView>
        <a-card title="列表">
            <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="false" bordered
                :scroll="{ x: true }">
            </a-table>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'

const DISPLAY = [
    { key: 'direct_rate', name: '直推奖比例' },
    { key: 'match_rate', name: '对碰奖比例' },
    { key: 'manage_rate', name: '管理奖比例' },
    { key: 'manage_generations', name: '管理奖代数' },
    { key: 'min_withdraw_amount', name: 'USDT 单笔最低提现' },
    { key: 'min_withdraw_amount_ispay', name: 'ISPAY 单笔最低提现' },
    { key: 'withdraw_fee_rate', name: 'USDT 提现手续费' },
    { key: 'withdraw_fee_rate_ispay', name: 'ISPAY 提现手续费' },
    { key: 'withdraw_daily_limit', name: 'USDT 每日提现上限' },
    { key: 'withdraw_daily_limit_ispay', name: 'ISPAY 每日提现上限' },
    { key: 'ispay_price', name: 'ISPAY 测试现价' },
    { key: 'overflow_clear_hours', name: '冻结清除时间' },
    { key: 'withdraw_enabled', name: '提现开关' },
]

export default {
    name: 'config',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: '名称',
                    dataIndex: 'name',
                },
                {
                    title: '值',
                    dataIndex: 'value',
                },
                {
                    title: '操作',
                    key: 'action',
                    fixed: 'right',
                    width: 110,
                    customRender: (v) => {
                        return <a-button type="primary" onClick={() => {
                            this.config_update(v.id);
                        }}>修改</a-button>
                    },
                },
            ],
        }
    },
    methods: {
        getList() {
            this.loading = true
            Gai.config().then((res) => {
                const byKey = {}
                ;(res.config || []).forEach((row) => {
                    byKey[row.key] = row
                })
                this.data = DISPLAY.map((item, key) => {
                    const row = byKey[item.key]
                    if (!row) return null
                    return { ...row, name: item.name, key }
                }).filter(Boolean)
                this.loading = false
            }).catch(() => {
                this.loading = false
            })
        },
        config_update(id) {
            let value = ""
            this.$confirm({
                title: `修改`,
                content: (
                    <a-input style="margin-top:25px;" placeholder="请输入" onInput={(val) => {
                        value = val.target.value
                    }} />
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.config_update({ id, value }).then(() => {
                            resolve()
                            this.getList()
                        }).catch(() => {
                            reject()
                        })
                    })
                }
            })
        }
    },
}
</script>

<style scoped lang="less">
.inputGroup {
    >div {
        margin-bottom: 20px;
    }
}
</style>
