<template>
    <PageView>
        <a-card title="分红流水">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="账户地址" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.reason" style="width:100%" placeholder="收益类型"
                        @change="getListTwo">
                        <template v-for="(item, index) in Object.values(reasonType)">
                            <a-select-option :value="Object.keys(reasonType)[index]">{{item}}</a-select-option>
                        </template>
                    </a-select>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-button-group>
                        <a-button type="primary" :loading="loading" @click="getListTwo">确定筛选</a-button>
                    </a-button-group>
                </a-col>
            </a-row>
            <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="{ total, pageSize, current }"
                @change="changePagination" bordered :scroll="{ x: true }">
            </a-table>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'

const reasonType = {
    reward: '全部收益',
    static: '静态收益',
    dynamic: '动态收益',
    direct: '直推奖励',
    match: '对碰奖励',
    manage: '管理奖',
}

export default {
    name: 'ordersList',
    mixins: [listMixin],
    data() {
        return {
            reasonType,
            columns: [
                {
                    title: '时间',
                    dataIndex: 'createdAt',
                },
                {
                    title: '结算日',
                    dataIndex: 'settleDate',
                    customRender: (v) => v || '-',
                },
                {
                    title: '地址',
                    dataIndex: 'address',
                    customRender: (v) => v || '-',
                },
                {
                    title: '大类',
                    dataIndex: 'category',
                    customRender: (v) => v || '-',
                },
                {
                    title: '收益类型',
                    dataIndex: 'reason',
                    customRender: (v, row) => row.name || reasonType[v] || v || '-',
                },
                {
                    title: '到账U',
                    dataIndex: 'amount',
                },
                {
                    title: '到账ISPAY',
                    dataIndex: 'amountTwo',
                },
                {
                    title: '入账账户',
                    dataIndex: 'balanceName',
                    customRender: (v) => v || '-',
                },
                {
                    title: '来源订单',
                    dataIndex: 'orderSource',
                    customRender: (v, row) => {
                        if (v) return v
                        if (row.orderNo && row.orderAmount) return `${row.orderNo} / ${row.orderAmount}`
                        return row.orderNo || '-'
                    },
                },
                {
                    title: '来源地址',
                    dataIndex: 'sourceAddress',
                    customRender: (v) => v || '-',
                },
                {
                    title: '代数',
                    dataIndex: 'num',
                    customRender: (v) => v || '-',
                },
                {
                    title: '明细',
                    dataIndex: 'detail',
                    customRender: (v, row) => v || row.remark || '-',
                },
            ],
            searchData: {
                address: '',
                reason: 'reward',
            },
        }
    },
    methods: {
        getList() {
            this.loading = true
            Gai.reward_list({
                page: this.current,
                ...this.searchData
            }).then((res) => {
                this.data = (res.rewards || []).map((value, key) => {
                    return { ...value, key }
                })
                this.loading = false
                this.total = parseInt(res.count)
            })
        },
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
