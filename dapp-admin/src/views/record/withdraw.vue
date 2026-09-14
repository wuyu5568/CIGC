<template>
    <PageView>
        <a-card title="列表">
            <a slot="extra">提币总量: {{ num }}</a>
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.username" placeholder="请输入用户地址" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="请输入地址" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-button-group>
                        <a-button type="primary" :loading="loading" @click="getListTwo">确定筛选</a-button>
                    </a-button-group>
                </a-col>
            </a-row>
            <a-table
                :loading="loading"
                :columns="columns"
                :dataSource="data"
                :pagination="{ total, pageSize, showSizeChanger, current }"
                @change="changePagination"
                bordered
                :scroll="{ x: true }"
            >
            </a-table>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Log from '../../api/Log'
import listMixin from '../mixin/listMixin'
export default {
    name: 'withdraw',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: 'ID',
                    dataIndex: 'id',
                    customRender: (v) => <a>{v}</a>,
                },
                {
                    title: '用户名',
                    dataIndex: 'username',
                },
                {
                    title: '数量',
                    dataIndex: 'all_money',
                },
                {
                    title: '类型',
                    dataIndex: 'type',
                    customRender: (v) => v === "1"?`USDT`:`TDH`
                },
                {
                    title: '实际到账数量',
                    dataIndex: 'money',
                },
                {
                    title: '地址',
                    dataIndex: 'address',
                },
                {
                    title: '状态',
                    dataIndex: 'status',
                    customRender: (v) => {
                        let status = {
                            2: { name: '被驳回', color: 'error' },
                            0: { name: '未审核', color: 'warning' },
                            1: { name: '已通过', color: 'success' },
                        }
                        return <a-badge status={status[v].color} text={status[v].name} />
                    },
                },
                {
                    title: '时间',
                    dataIndex: 'add_time',
                    customRender: (v) => this.timeOne(v),
                },
                {
                        title: '操作',
                        key: 'action',
                        fixed: 'right',
                        width: 110,
                        customRender: (v) => {
                            return (
                                <a-dropdown>
                                    <a-menu slot="overlay">
                                        <a-menu-item onClick={() => {
                                            this.changeStatus(v.id,"通过",true)
                                        }} disabled={v.status!=="0"}>通过
                                        </a-menu-item>
                                        <a-menu-item onClick={() => {
                                            this.changeStatus(v.id,"驳回",false)
                                        }} disabled={v.status!=="0"}>驳回
                                        </a-menu-item>
                                    </a-menu>
                                    <a-button>操作 <a-icon type="down"/></a-button>
                                </a-dropdown>
                            )
                        }
                    }
            ],
            searchData: {
                username: '',
                address: '',
            },
            num: 0,
        }
    },
    methods: {
        getList() {
            this.loading = true
            Log.getWithdrawLog({
                page: this.current,
                num: this.pageSize,
                ...this.searchData,
            }).then((res) => {
                this.data = res.data.map((value, key) => {
                    return { ...value, key }
                })
                this.num = res.num
                this.loading = false
                this.total = parseInt(res.count)
            })
        },
        changeStatus(id,title,status){
                this.$confirm({
                    title: `${title}`,
                    content: `确定要${title}此提现吗?`,
                    centered: true,
                    onOk: () => {
                        return new Promise((resolve, reject) => {
                            Log[status?`withdraw_adopt`:`withdraw_refuse`]({ id }).then(res => {
                                resolve()
                                this.getList()
                            }).catch(res => {
                                reject()
                            })
                        })
                    }
                })
            },
    },
}
</script>

<style scoped lang="less">
.inputGroup {
    > div {
        margin-bottom: 20px;
    }
}
</style>