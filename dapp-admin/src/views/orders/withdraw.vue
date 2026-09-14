<template>
    <PageView>
        <a-card title="列表">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="账户" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.withDrawType" style="width:100%" placeholder="类型"
                        @change="getListTwo">
                        <a-select-option value="USDT">USDT</a-select-option>
                        <a-select-option value="RAW_NEW">ISPAY</a-select-option>
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
export default {
    name: 'withdraw',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: '账户',
                    dataIndex: 'address',
                },
                // {
                //     title: '类型',
                //     dataIndex: 'type',
                //     customRender: (v) => v === "usdt" ? `模式1` : "模式2"
                // },
                // {
                //     title: '到账金额',
                //     dataIndex: 'amount',
                // },
                {
                    title: '到账金额',
                    dataIndex: 'relAmount',
                },
                {
                    title: '提现金额',
                    dataIndex: 'amount',
                },
                // {
                //     title: '状态',
                //     dataIndex: 'status',
                //     customRender: (v) => ({
                //         "success": "成功",
                //         "doing": "正在处理",
                //     }[v])
                // },
                {
                    title: '创建时间',
                    dataIndex: 'createdAt',
                },
                // {
                //     title: '操作',
                //     key: 'action',
                //     fixed: 'right',
                //     width: 110,
                //     customRender: (v) => {
                //         return <a-button type="primary" disabled={v.status !== "doing"} onClick={() => this.withdraw(v.id)}>通过</a-button>
                //     },
                // },
            ],
            searchData: {
                address: '',
                withDrawType: undefined,
            },
        }
    },
    methods: {
        getList() {
            this.loading = true
            Gai.withdraw_list({
                page: this.current,
                ...this.searchData
            }).then((res) => {
                this.data = res.withdraw.map((value, key) => {
                    return { ...value, key }
                })
                this.loading = false
                this.total = parseInt(res.count)
            })
        },
        withdraw(id) {
            this.$confirm({
                title: `通过提示`,
                content: `确定要通过此次提现吗?`,
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.withdraw_pass({ id }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
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