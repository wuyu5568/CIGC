<template>
    <PageView>
        <a-card title="列表">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.uid" placeholder="请输入用户ID" @keyup.enter="getListTwo"/>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.id" placeholder="请输入ID" @keyup.enter="getListTwo"/>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.node_id" placeholder="请输入上级ID" @keyup.enter="getListTwo"/>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.order_no" placeholder="请输入订单号" @keyup.enter="getListTwo"/>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.sort" style="width:100%" placeholder="类型筛选"
                              @change="getListTwo">
                        <a-select-option :value="value.dataIndex" v-for="(value,key) in config">{{value.title}}</a-select-option>
                    </a-select>
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
                :pagination="{total,pageSize,showSizeChanger,current}"
                @change="changePagination"
                bordered :scroll="{x:true}">
            </a-table>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
    import Member from '../../api/Member'
    import listMixin from '../mixin/listMixin'

    const config = [
        {
            title: '累计收益',
            dataIndex: 'all_profit'
        },
        {
            title: '节点收益',
            dataIndex: 'node_profit'
        },
        {
            title: '推广收益',
            dataIndex: 'team_profit'
        },
        {
            title: '节点团队',
            dataIndex: 'node_team_num'
        },
        {
            title: '今日奖励',
            dataIndex: 'all_profit_today'
        },
        {
            title: '节点奖励',
            dataIndex: 'node_profit_today'
        },
        {
            title: '推广奖励',
            dataIndex: 'team_profit_today'
        },
        {
            title: '推广团队',
            dataIndex: 'team_num'
        }
    ]
    export default {
        name: 'superNodeList',
        mixins: [listMixin],
        data () {
            return {
                config,
                columns: [
                    {
                        title: 'ID',
                        dataIndex: 'id',
                        customRender: (v) => <a>{v}</a>
                    },
                    {
                        title: '用户ID',
                        dataIndex: 'uid'
                    },
                    {
                        title: '邀请码',
                        dataIndex: 'invite_code'
                    },
                    {
                        title: '用户名称',
                        dataIndex: 'username'
                    },
                    {
                        title: '是否为超级节点',
                        dataIndex: 'is_super',
                        customRender: (v) => {
                            let status = {
                                0: { name: '否', color: 'error' },
                                [null]: { name: '否', color: 'error' },
                                1: { name: '是', color: 'success' },
                            }
                            return <a-badge status={status[v].color} text={status[v].name}/>
                        }
                    },
                    {
                        title: '伞下节点数量',
                        dataIndex: 'super_node_num'
                    },
                    {
                        title: '节点池',
                        dataIndex: 'super_pool'
                    },
                    ...config,
                    {
                        title: '订单号',
                        dataIndex: 'order_no'
                    },
                    {
                        title: '状态',
                        dataIndex: 'status',
                        customRender: (v) => {
                            let status = {
                                0: { name: '未完成', color: 'error' },
                                1: { name: '已完成', color: 'success' },
                                2: { name: '已结束', color: 'success' }
                            }
                            return <a-badge status={status[v].color} text={status[v].name}/>
                        }
                    },
                    {
                        title: '成交时间',
                        dataIndex: 'deal_time',
                        customRender: (v) => this.timeOne(v)
                    }
                ],
                searchData: {
                    uid: '',
                    id: '',
                    node_id: '',
                    order_no: '',
                    sort: undefined
                }
            }
        },
        methods: {
            getList () {
                this.loading = true
                Member.superNodeList({
                    page: this.current,
                    num: this.pageSize,
                    ...this.searchData
                }).then(res => {
                    this.data = res.data.map((value, key) => {
                        return { ...value, key }
                    })
                    this.loading = false
                    this.total = parseInt(res.count)
                })
            }
        }
    }
</script>

<style scoped lang="less">
    .inputGroup {
        > div {
            margin-bottom: 20px;
        }
    }
</style>