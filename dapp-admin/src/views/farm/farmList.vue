<template>
    <PageView>
        <a-card title="列表">
            <a-table
                :loading="loading"
                :columns="columns"
                :dataSource="data"
                :pagination="false"
                bordered :scroll="{x:true}">
            </a-table>
        </a-card>
        <!--修改-->
        <a-modal
            title="修改"
            :visible="visible"
            :confirm-loading="confirmLoading"
            centered
            :closable="false"
            :maskClosable="false"
            width="370px"
            @ok="start"
            @cancel="visible = false"
        >
            <a-form :form="form">
                <a-form-item label="开始时间">
                    <a-date-picker style="width:100%" v-decorator="['start_time', {rules: [{ type: 'object', required: true, message: '请选择开始时间' }]}]" show-time format="YYYY-MM-DD HH:mm:ss"/>
                </a-form-item>
                <a-form-item label="共有">
                    <a-input v-decorator="[`total_supply`, { rules: [{ required: true, message: '请输入' }]}]" placeholder="请输入"/>
                </a-form-item>
                <a-form-item label="剩余">
                    <a-input v-decorator="[`supply`, { rules: [{ required: true, message: '请输入' }]}]" placeholder="请输入"/>
                </a-form-item>
            </a-form>
        </a-modal>
    </PageView>
</template>

<script type="text/jsx">
    import moment from 'moment';
    import Ore from '../../api/Ore'
    import listMixin from '../mixin/listMixin'
    export default {
        name: 'farmList',
        mixins:[listMixin],
        data(){
            return{
                columns: [
                    {
                        title: 'ID',
                        dataIndex: 'id',
                    },
                    {
                        title: '名称',
                        dataIndex: 'name',
                        customRender: (v) => <a>{v}</a>
                    },
                    {
                        title: '共有',
                        dataIndex: 'total_supply',
                    },
                    {
                        title: '剩余',
                        dataIndex: 'supply',
                    },
                    {
                        title: '总质押',
                        dataIndex: 'total',
                    },
                    {
                        title: '开始时间',
                        dataIndex: 'start_time',
                        customRender: (v) => Number(v)?this.timeOne(v):"未设置"
                    },
                    {
                        title: '结束时间',
                        dataIndex: 'over_time',
                        customRender: (v) => Number(v)?this.timeOne(v):"未设置"
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
                                            this.id = v.id
                                            this.visible = true
                                            this.$nextTick(()=>this.form.setFieldsValue({
                                                [`start_time`]:moment(this.timeOne(v.start_time),"YYYY-MM-DD HH:mm:ss"),
                                                [`total_supply`]:v.total_supply,
                                                [`supply`]:v.supply,
                                            }))
                                        }}>修改
                                        </a-menu-item>
                                    </a-menu>
                                    <a-button>操作 <a-icon type="down"/></a-button>
                                </a-dropdown>
                            )
                        }
                    }
                ],
                visible:false,
                confirmLoading:false,
                form: this.$form.createForm(this),
                id:undefined
            }
        },
        methods: {
            getList () {
                this.loading = true
                Ore.oreList().then(res => {
                    this.data = res.data.map((value, key) => {
                        return { ...value, key }
                    })
                    this.loading = false
                })
            },
            start(){
                this.form.validateFields((err, values) => {
                    if (!err) {
                        let data = {...values}
                        data.start_time = Date.parse(data.start_time)/1000
                        this.confirmLoading = true
                        Ore.start ({...data,id:this.id }).then(res => {
                            this.confirmLoading = false
                            this.visible = false
                            this.getList()
                        }).catch(res=>{
                            this.confirmLoading = false
                        })
                    }
                });
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