<template>
    <PageView>
        <a-card title="列表">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.username" placeholder="请输入钱包地址" @keyup.enter="getListTwo"/>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.type" style="width:100%" placeholder="类型筛选"
                              @change="getListTwo">
                        <a-select-option :value="key" v-for="(value,key) in typeList" :key="key">{{value}}</a-select-option>
                    </a-select>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.sz_type" style="width:100%" placeholder="收支类型"
                              @change="getListTwo">
                        <a-select-option value="1">收入</a-select-option>
                        <a-select-option value="2">支出</a-select-option>
                    </a-select>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.money_type" style="width:100%" placeholder="币种类型"
                              @change="getListTwo">
                        <a-select-option value="1">USDT</a-select-option>
                        <a-select-option value="2">TDH</a-select-option>
                        <a-select-option value="3">DFCC</a-select-option>
                        <a-select-option value="4">TDHT</a-select-option>
                    </a-select>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-button-group>
                        <a-button type="primary" :loading="loading" @click="getListTwo">确定筛选</a-button>
                    </a-button-group>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    金额:{{money}}
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
    import Finance from '../../api/Finance'
    import listMixin from '../mixin/listMixin'
    export default {
        name: 'finance',
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
                        title: '内容',
                        dataIndex: 'content',
                    },
                    {
                        title: '种类',
                        dataIndex: 'type_name',
                    },
                    {
                        title: '币种',
                        dataIndex: 'money_type',
                        customRender: (v) => ({
                            1:"USDT",
                            2:"TDH",
                            3:"DFCC",
                            4:"TDHT",
                        })[v]
                    },
                    {
                        title: '数量',
                        dataIndex: 'money',
                    },
                    {
                        title: '收支类型',
                        dataIndex: 'sz_type',
                        customRender: (v) =>v==="1"?`收入`:`支出`
                    },
                    {
                        title: '时间',
                        dataIndex: 'add_time',
                        customRender: (v) => this.timeOne(v)
                    },
                ],
                searchData:{
                    username:"",
                    type: undefined,
                    sz_type: undefined,
                    money_type: undefined,
                },
                typeList:{},
                money:""
            }
        },
        methods: {
            getList () {
                this.loading = true
                Finance.getFinanceAll({
                    page: this.current,
                    num: this.pageSize,
                    ...this.searchData
                }).then(res => {
                    this.data = res.data.map((value, key) => {
                        return { ...value, key }
                    })
                    this.typeList = res.finance_type;
                    this.money = res.money
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