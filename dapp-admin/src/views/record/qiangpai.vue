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
    import Project from '../../api/Project'
    import listMixin from '../mixin/listMixin'
    export default {
        name: 'qiangpai',
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
                        title: '商品名',
                        dataIndex: 'project_name',
                    },
                    {
                        title: '场次',
                        dataIndex: 'type_name',
                    },
                    {
                        title: '保证金',
                        dataIndex: 'bond',
                    },
                    {
                        title: '状态',
                        dataIndex: 'status',
                        customRender: (v) => {
                            let status = {
                                "2": { name: '抢拍成功', color: 'success' },
                                "0": { name: '匹配中', color: 'warning' },
                                "1": { name: '交易中', color: 'warning' },
                                "3": { name: '已转拍', color: 'success' },
                                "-2": { name: '抢拍失败', color: 'error' },
                                "-3": { name: '已失效', color: 'error' },
                            }
                            return <a-badge status={status[v].color} text={status[v].name}/>
                        }
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
                Project.getProjectLog({
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