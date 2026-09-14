<template>
    <PageView>
        <a-card title="列表">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.username" placeholder="请输入钱包地址" @keyup.enter="getListTwo"/>
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
        name: 'node',
        mixins:[listMixin],
        data(){
            return{
                columns: [
                    {
                        title: '用户名',
                        dataIndex: 'username',
                        customRender: (v) => <a>{v}</a>
                    },
                    {
                        title: '总数量',
                        dataIndex: 'all_money',
                    },
                    {
                        title: '剩余数量',
                        dataIndex: 'money',
                    },
                    {
                        title: '时间',
                        dataIndex: 'add_time',
                        customRender: (v) => this.timeOne(v)
                    },
                ],
                searchData:{
                    username:"",
                },
            }
        },
        methods: {
            getList () {
                this.loading = true
                Log.getNodeLog({
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