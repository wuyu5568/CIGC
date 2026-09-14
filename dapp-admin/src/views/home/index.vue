<template>
    <PageView>
        <a-card class="cardCon" title="数据" :loading="loading">
            <template slot="extra">
                <span style="margin-right:12px">{{ settleHint }}</span>
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
                const today = day ? (done ? `${day} 已结算` : `${day} 未结算`) : ''
                this.settleHint = next ? `${today}；测试将结算 ${next}` : today
            }).catch(() => {
                this.settleHint = ''
            })
        },
        runSettle() {
            this.$confirm({
                title: '测试日结算',
                content: '将结算下一日（静态释放、补漏动态、按封顶解冻）。冻结批次按该日 0:00 判断是否已满冻结日+4天，到期只清零、不转入可提现。不是重跑今天。仅测试用。',
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
