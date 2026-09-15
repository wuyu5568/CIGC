<template>
    <PageView>
        <a-card class="cardCon" title="数据" :loading="loading">
            <template slot="extra">
                <span style="margin-right:12px">{{ settleHint }}</span>
                <a-button style="margin-right:8px" type="danger" :loading="clearing" @click="clearTestData">清除测试数据</a-button>
                <a-button style="margin-right:8px" :loading="resetting" @click="resetTestDay">重置测试日</a-button>
                <a-button type="primary" :loading="settling" @click="runSettle">测试日结算</a-button>
            </template>
            <a-row :gutter="0">
                <a-col v-for="item in cards" :key="item.label" :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>{{ item.label }}:</a> {{ display(item.key) }}
                        </div>
                    </a-card-grid>
                </a-col>
            </a-row>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
export default {
    name: 'home',
    data() {
        return {
            loading: true,
            settling: false,
            resetting: false,
            clearing: false,
            data: {},
            settleHint: '',
            cards: [
                { label: '注册人数', key: 'totalUserR' },
                { label: '激活总人数', key: 'totalUser' },
                { label: '今日注册人数', key: 'todayUserR' },
                { label: '今日激活人数', key: 'todayUser' },
                { label: '总充值U', key: 'depositTotal' },
                { label: '剩余充值余额', key: 'rechargeRemain' },
                { label: '管理后台加U', key: 'adminRecharge' },
                { label: '今日充值U', key: 'todayDeposit' },
                { label: '购买订单总数', key: 'orderCount' },
                { label: '今日购买订单数量', key: 'todayOrderCount' },
                { label: '认购总金额', key: 'buyTotal' },
                { label: '今日认购金额', key: 'todayBuy' },
                { label: '今日静态释放', key: 'todayOne' },
                { label: '今日USDT动态', key: 'todayTwo' },
                { label: '今日ispay动态', key: 'todayIspayDyn' },
                { label: '累计静态产值', key: 'totalStatic' },
                { label: '累计动态产值', key: 'totalDynamic' },
                { label: '全网可提USDT', key: 'balanceUsdt' },
                { label: '全网可提ispay', key: 'balanceIspay' },
                { label: '全网收益冻结USDT', key: 'lockBalance' },
                { label: '全网收益冻结ispay', key: 'lockIspay' },
                { label: '全网提现冻结USDT', key: 'frozen' },
                { label: '全网提现冻结ispay', key: 'frozenIspay' },
                { label: '今日提现USDT', key: 'todayWithdraw' },
                { label: '总提USDT', key: 'totalWithdraw' },
                { label: '今日ispay提现', key: 'todayWithdrawIspay' },
                { label: '总ispay提现', key: 'totalWithdrawIspay' },
            ],
        }
    },
    activated() {
        this.getList()
        this.getSettle()
    },
    methods: {
        display(key) {
            const v = this.data[key]
            if (v === 0 || v === '0') return v
            return v || 0
        },
        getList() {
            this.loading = true
            Gai.all().then(res => {
                this.data = res
                this.loading = false
            })
        },
        getSettle() {
            Gai.settle_status().then(res => {
                const day = res.settle_today || ''
                const done = res.settle_today_done
                const next = res.settle_next_test || ''
                const biz = res.settle_business_date || ''
                const today = day ? (done ? `${day} 已结算` : `${day} 未结算`) : ''
                let hint = today
                if (biz && biz !== day) {
                    hint = hint ? `${hint}；业务日 ${biz}` : `业务日 ${biz}`
                }
                if (next) {
                    hint = hint ? `${hint}；下次结算 ${next}` : `下次结算 ${next}`
                }
                this.settleHint = hint
            }).catch(() => {
                this.settleHint = ''
            })
        },
        runSettle() {
            this.$confirm({
                title: '测试日结算',
                content: '将结算下一日（静态释放、补漏动态、按封顶解冻）。结算完成后，新产生的冻结按该测试日记账：到期为冻结日+4天 00:00，只清零、不转入可提现。例如点到 16 号后再产生的冻结，要再点到 20 号才清；点到 16 号之前产生的冻结，点到 19 号就清。',
                centered: true,
                onOk: () => {
                    this.settling = true
                    return Gai.settle({ force: 1 }).then((res) => {
                        const date = res.settle_date || ''
                        const burned = res.overflow_cleared || 0
                        this.$message.success(date
                            ? `日结完成 ${date}，冻结到期清除 ${burned} 笔`
                            : '日结完成')
                        this.getList()
                        this.getSettle()
                    }).finally(() => {
                        this.settling = false
                    })
                },
            })
        },
        resetTestDay() {
            this.$confirm({
                title: '重置测试日',
                content: '将删除今天之后的测试日结记录，把测试日拨回今天。下一次「测试日结算」将结算明天。不恢复已清零的冻结，不改余额。',
                centered: true,
                onOk: () => {
                    this.resetting = true
                    return Gai.settle_reset().then((res) => {
                        const today = res.settle_today || ''
                        const next = res.settle_next_test || ''
                        const n = res.deleted || 0
                        this.$message.success(`测试日已重置为 ${today}，已删 ${n} 条；下次将结算 ${next}`)
                        this.getList()
                        this.getSettle()
                    }).finally(() => {
                        this.resetting = false
                    })
                },
            })
        },
        clearTestData() {
            this.$confirm({
                title: '清除测试数据',
                content: '将删除订单、流水、冻结、充值入账、提现和日结记录，并把所有账户余额、已支付、日封顶归零。用户地址、邀请关系和双轨安置会保留。此操作不可恢复。',
                centered: true,
                okType: 'danger',
                onOk: () => {
                    this.clearing = true
                    return Gai.test_data_clear().then((res) => {
                        const users = res.users_kept || 0
                        const orders = res.orders_cleared || 0
                        this.$message.success(`已清测试数据：保留 ${users} 个账户，删除 ${orders} 笔订单`)
                        this.getList()
                        this.getSettle()
                    }).finally(() => {
                        this.clearing = false
                    })
                },
            })
        },
    }
}
</script>

<style scoped lang="less">
.cardCon {
    /deep/ .ant-card-body {
        padding: 0 !important;
    }

    .ant-card-grid {
        padding: 24px 15px;
        width: 100%;
        display: flex;
        align-items: center;
        justify-content: space-between;

        i {
            font-size: 20px;
            color: #ffffff;
            padding: 10px;
            border-radius: 8px;
            background: #1890ff;
        }
    }
}

.inputGroup {
    >div {
        margin: 20px;
    }
}
</style>
