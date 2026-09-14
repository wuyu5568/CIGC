<template>
    <PageView>
        <a-card title="列表">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.username" placeholder="请输入用户名" @keyup.enter="getListTwo"/>
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
        name: 'exchange',
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
                        title: '用户名',
                        dataIndex: 'username',
                    },
                    {
                        title: '用户ID',
                        dataIndex: 'uid',
                    },
                    {
                        title: '兑换数量',
                        dataIndex: 'money',
                    },
                    {
                        title: '实际到账数量',
                        dataIndex: 'in_money',
                    },
                    {
                        title: '类型',
                        dataIndex: 'type',
                        customRender: (v) =>{
                            return <div>{v === "1"?`USDT`:`TDH`} <a><a-icon type="swap" /></a> {v === "2"?`USDT`:`TDH`}</div>
                        }
                    },
                    {
                        title: '时间',
                        dataIndex: 'add_time',
                        customRender: (v) => this.timeOne(v)
                    },
                ],
                searchData: {
                    username: "",
                },
                num1:0,
                num2:0,
                today_num1:0,
                today_num2:0,
            }
        },
        methods: {
            getList () {
                this.loading = true
                Log.getExchangelog({
                    page: this.current,
                    num: this.pageSize,
                    ...this.searchData
                }).then(res => {
                    this.data = res.data.map((value, key) => {
                        return { ...value, key }
                    })
                    this.num1 = res.num1||0
                    this.num2 = res.num2||0
                    this.today_num1 = res.today_num1||0
                    this.today_num2 = res.today_num2||0
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