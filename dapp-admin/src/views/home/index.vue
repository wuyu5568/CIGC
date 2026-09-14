<template>
    <PageView>
        <a-card class="cardCon" title="数据" :loading="loading">
            <template slot="extra">
                <span style="margin-right:12px">{{ settleHint }}</span>
                <a-button type="primary" :loading="settling" @click="runSettle">测试日结算</a-button>
            </template>
            <a-row :gutter="0">
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>注册人数:</a> {{ data.totalUserR || 0 }}
                        </div>
                    </a-card-grid>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>激活总人数:</a> {{ data.totalUser || 0 }}
                        </div>
                    </a-card-grid>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>今日注册人数:</a> {{ data.todayUserR || 0 }}
                        </div>
                    </a-card-grid>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>今日激活人数:</a> {{ data.todayUser || 0 }}
                        </div>
                    </a-card-grid>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>认购总数:</a> {{ data.buyTotal || 0 }}
                        </div>
                    </a-card-grid>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>今日认购总数:</a> {{ data.todayBuy || 0 }}
                        </div>
                    </a-card-grid>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>全网可提U:</a> {{ data.balanceUsdt || 0 }}
                        </div>
                    </a-card-grid>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>今日静态:</a> {{ data.todayOne || 0 }}
                        </div>
                    </a-card-grid>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>今日动态:</a> {{ data.todayTwo || 0 }}
                        </div>
                    </a-card-grid>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>今日动静态:</a> {{ data.todayThree || 0 }}
                        </div>
                    </a-card-grid>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>总动静态累计:</a> {{ data.totalReward || 0 }}
                        </div>
                    </a-card-grid>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>今日提现:</a> {{ data.todayWithdraw || 0 }}
                        </div>
                    </a-card-grid>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>总提现:</a> {{ data.totalWithdraw || 0 }}
                        </div>
                    </a-card-grid>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="6">
                    <a-card-grid>
                        <a-icon type="team" />
                        <div>
                            <a>ispay总数:</a> {{ data.totalIspay || 0 }}
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
            data: {},
            settleHint: '',
        }
    },
    activated() {
        this.getList()
        this.getSettle()
    },
    methods: {
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
                content: '将结算下一日收益（静态释放、补漏直推/对碰/管理奖、按封顶解冻），不是重跑今天。仅测试用。',
                centered: true,
                onOk: () => {
                    this.settling = true
                    return Gai.settle({ force: 1 }).then((res) => {
                        const date = res.settle_date || ''
                        this.$message.success(date ? `日结完成 ${date}` : '日结完成')
                        this.getList()
                        this.getSettle()
                    }).finally(() => {
                        this.settling = false
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