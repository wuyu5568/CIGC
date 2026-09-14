<template>
    <PageView>
        <a-card title="列表">
            <a slot="extra" v-if="searchData.type&&!loading">总数: {{num}}</a>
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.type" style="width:100%" placeholder="选择银行"
                              @change="getListTwo">
                        <a-select-option :value="value.id" v-for="(value,key) in typeList">{{value.name}}</a-select-option>
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
    import Bank from '../../api/Bank'
    import listMixin from '../mixin/listMixin'
    export default {
        name: 'liveSaveList',
        mixins:[listMixin],
        data(){
            return{
                columns: [
                    {
                        title: '名称',
                        dataIndex: 'name',
                        customRender: (v) => <a>{v}</a>
                    },
                    {
                        title: '用户名',
                        dataIndex: 'username',
                    },
                    {
                        title: '数量',
                        dataIndex: 'num',
                    },
                    {
                        title: '可提取',
                        dataIndex: 'money',
                    },
                    {
                        title: '总收益',
                        dataIndex: 'all_money',
                    },
                ],
                searchData:{
                    type: undefined,
                },
                typeList:[],
                num:0
            }
        },
        created() {
            this.getAllType()
        },
        methods: {
            getAllType(){
                Bank.bankList().then(res=>this.typeList = res.data)
            },
            getList () {
                this.loading = true
                Bank.getBanklog({
                    page: this.current,
                    num: this.pageSize,
                    ...this.searchData
                }).then(res => {
                    this.data = res.data.map((value, key) => {
                        return { ...value, key }
                    })
                    this.num = res.num||0
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