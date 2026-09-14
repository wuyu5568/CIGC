<template>
    <PageView>
        <a-card title="列表">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.uid" placeholder="请输入用户ID" @keyup.enter="getListTwo"/>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.node_id" placeholder="请输入节点ID" @keyup.enter="getListTwo"/>
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
    import Log from '../../api/Log'
    import listMixin from '../mixin/listMixin'
    export default {
        name: 'ore',
        mixins:[listMixin],
        data(){
            return{
                columns: [
                    {
                        title: 'ID',
                        dataIndex: 'id',
                        customRender: (v) => <a>{v}</a>
                    },
                    {
                        title: '用户ID',
                        dataIndex: 'uid',
                    },
                    {
                        title: '总数量',
                        dataIndex: 'num',
                    },
                    {
                        title: '已完成数量',
                        dataIndex: 'over_num',
                        customRender:(v)=><a-badge status="success" text={v}/>
                    },
                    {
                        title: '每日挖矿数量',
                        dataIndex: 'day_num',
                    },
                    {
                        title: '状态',
                        dataIndex: 'status',
                        customRender: (v) => {
                            let status = {
                                0: { name: '未完成', color: 'error' },
                                1: { name: '已完成', color: 'success' },
                            }
                            return <a-badge status={status[v].color} text={status[v].name}/>
                        }
                    },
                    {
                        title: '开始时间',
                        dataIndex: 'add_time',
                        customRender: (v) => this.timeOne(v)
                    },
                    {
                        title: '结束时间',
                        dataIndex: 'over_time',
                        customRender: (v) => this.timeOne(v)
                    },
                ],
                searchData: {
                    uid: "",
                    node_id: "",
                },
            }
        },
        methods: {
            getList () {
                this.loading = true
                Log.getOrelog({
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
            },
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