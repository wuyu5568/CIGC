<template>
    <PageView>
        <a-card title="列表">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.username" placeholder="请输入钱包地址" @keyup.enter="getListTwo"/>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.money" placeholder="金额" @keyup.enter="getListTwo"/>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.project_id" style="width:100%" placeholder="选择商品"
                              @change="getListTwo">
                        <a-select-option :value="value.id" v-for="(value,key) in goodsList" :key="key">{{value.name}}</a-select-option>
                    </a-select>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.type" style="width:100%" placeholder="选择场次"
                              @change="getListTwo">
                        <a-select-option :value="value.id" v-for="(value,key) in typeList" :key="key">{{value.name}}</a-select-option>
                    </a-select>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.status" style="width:100%" placeholder="选择状态"
                              @change="getListTwo">
                        <a-select-option value="-1">失效</a-select-option>
                        <a-select-option value="0">待交易</a-select-option>
                        <a-select-option value="1">交易中</a-select-option>
                        <a-select-option value="2">交易成功</a-select-option>
                        <a-select-option value="-2">已赔付</a-select-option>
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
    import Project from '../../api/Project'
    import listMixin from '../mixin/listMixin'
    export default {
        name: 'sale',
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
                        title: '金额',
                        dataIndex: 'money',
                    },
                    {
                        title: '状态',
                        dataIndex: 'status',
                        customRender: (v) => {
                            let status = {
                                "-1": { name: '失效', color: 'error' },
                                "0": { name: '待交易', color: 'warning' },
                                "1": { name: '交易中', color: 'warning' },
                                "2": { name: '交易成功', color: 'success' },
                                "-2": { name: '已赔付', color: 'success' },
                            }
                            return <a-badge status={status[v].color} text={status[v].name}/>
                        }
                    },
                    {
                        title: '时间',
                        dataIndex: 'add_time',
                        customRender: (v) => this.timeOne(v)
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
                                            this.dele(v.id)
                                        }} disabled={v.status!=="0"}>删除
                                        </a-menu-item>
                                    </a-menu>
                                    <a-button>操作 <a-icon type="down"/></a-button>
                                </a-dropdown>
                            )
                        }
                    }
                ],
                searchData:{
                    username:"",
                    money:"",
                    project_id:undefined,
                    type:undefined,
                    status:undefined
                },
                goodsList:[],
                typeList:[],
            }
        },
        created(){
            Project.getAllProject().then((res) => {
                this.goodsList = res.data;
            })
            Project.getProjectType().then((res) => {
                this.typeList = res.data;
            })
        },
        methods: {
            getList () {
                this.loading = true
                Project.getSaleList({
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
            dele(id){
                this.$confirm({
                    title: `删除`,
                    content: `确定要删除吗?`,
                    centered: true,
                    onOk: () => {
                        return new Promise((resolve, reject) => {
                            Project.deleteSaleLog({ id }).then(res => {
                                resolve()
                                this.getList()
                            }).catch(res => {
                                reject()
                            })
                        })
                    }
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