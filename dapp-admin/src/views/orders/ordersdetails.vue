<template>
    <PageView>
        <a-spin :spinning="loading">
            <a-card title="详情">
                <div class="content">
                    <div v-for="(v, k) in detailsList" :key="k">
                        <a class="name">{{ v.name }}:</a>
                        <div class="data">{{ v.data }}</div>
                    </div>
                    <!-- 申诉中 -->
                    <template v-if="details.status === `3`">
                        <div>
                            <a class="name">申诉内容:</a>
                            <div class="data">{{ details.t_order.coutent }}</div>
                        </div>
                        <div>
                            <a class="name">申诉图片:</a>
                            <img class="appeal" :src="imgUrl + details.t_order.image" />
                        </div>
                    </template>
                </div>
                <a-button-group class="btn-container">
                    <!-- <a-button type="primary" v-if="!(details.status === `2`)">买家收款方式</a-button> -->
                    <!-- <a-button type="primary" v-if="!(details.status === `2`)">卖家收款方式</a-button> -->
                    <a-button type="primary" v-if="details.status === `3`" @click="appeal(true)">申诉通过</a-button>
                    <a-button type="primary" v-if="details.status === `3`" @click="appeal(false)">申诉驳回</a-button>
                </a-button-group>
            </a-card>
        </a-spin>
    </PageView>
</template>
<script type="text/jsx">
import Order from '../../api/Order'
export default {
    name: 'ordersdetails',
    data() {
        return {
            loading: true,
            details: {
                buy_bank: [],
                sell_bank: [],
                t_order: {},
            },
            imgUrl: projectUrl,
        }
    },
    computed: {
        detailsList() {
            return [
                { name: '订单号', data: this.details.order_no },
                { name: '买方', data: this.details.buy_username },
                { name: '卖方', data: this.details.sell_username },
                { name: '名称', data: this.details.project_name },
                { name: '金额', data: this.details.money },
                { name: '转拍时间', data: this.details.add_time ? this.timeOne(this.details.add_time) : `暂无` },
                { name: '打款时间', data: this.details.pay_time ? this.timeOne(this.details.pay_time) : `暂无` },
                {
                    name: '成交时间',
                    data: this.details.confirm_time ? this.timeOne(this.details.confirm_time) : `暂无`,
                },
                {
                    name: '状态',
                    data: (() => {
                        let config = {
                            0: '待付款',
                            1: '已付款',
                            2: '已完成',
                            3: '申诉中',
                            4: '已赔付',
                        }
                        return config[this.details.status] || `加载中...`
                    })(),
                },
            ]
        },
    },
    activated() {
        this.orderDetalis()
    },
    methods: {
        orderDetalis() {
            this.loading = true
            Order.orderDetalis({
                id: this.$route.query.id,
            }).then((res) => {
                this.details = res.order
                this.loading = false
            })
        },
        appeal(bool) {
            this.$confirm({
                title: bool ? `申诉通过` : `申诉驳回`,
                content: `确定要${bool ? `通过` : `驳回`}吗?`,
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Order[bool ? `appealAdopt` : `appealRefuse`]({
                            order_id: this.details.id,
                            id: this.details.t_order.id,
                        })
                            .then((res) => {
                                resolve()
                                this.orderDetalis()
                            })
                            .catch((res) => {
                                reject()
                            })
                    })
                },
            })
        },
    },
}
</script>
<style lang="less" scoped>
.content {
    > div {
        display: flex;
        align-items: center;
        font-size: 15px;
        padding: 10px 0px;
        font-weight: bold;
        .name {
            margin-right: 20px;
        }
        .appeal {
            width: 600px;
        }
    }
}
.btn-container {
    margin-top: 10px;
}
</style>