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
                <a-form-item label="apy">
                    <a-input v-decorator="[`apy`, { rules: [{ required: true, message: '请输入' }]}]" placeholder="请输入"/>
                </a-form-item>
                <a-form-item label="状态">
                    <a-radio-group v-decorator="[`status`]">
                        <a-radio value="0">
                            进行中
                        </a-radio>
                        <a-radio value="1">
                            已结束
                        </a-radio>
                    </a-radio-group>
                </a-form-item>
            </a-form>
        </a-modal>
    </PageView>
</template>

<script type="text/jsx">
    import Bank from '../../api/Bank'
    import listMixin from '../mixin/listMixin'
    export default {
        name: 'bankList',
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
                        title: 'apy',
                        dataIndex: 'apy',
                    },
                    {
                        title: '状态',
                        dataIndex: 'status',
                        customRender: (v) => {
                            let status = {
                                0: { name: '进行中', color: 'success' },
                                1: { name: '已结束', color: 'error' }
                            }
                            return <a-badge status={status[v].color} text={status[v].name}/>
                        }
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
                                                [`apy`]:v.apy,
                                                [`status`]:v.status,
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
                Bank.bankList().then(res => {
                    this.data = res.data.map((value, key) => {
                        return { ...value, key }
                    })
                    this.loading = false
                })
            },
            start(){
                this.form.validateFields((err, values) => {
                    if (!err) {
                        this.confirmLoading = true
                        Bank.start ({...values,id:this.id }).then(res => {
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