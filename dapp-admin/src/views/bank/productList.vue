<template>
    <PageView>
        <a-card title="列表">
            <a-button type="primary" slot="extra" @click="showDialog()">
                <a-icon type="plus"/>
                添加
            </a-button>
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
            :title="!id?`新增`:`修改`"
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
                <a-form-item label="名称">
                    <a-input v-decorator="[`name`, { rules: [{ required: true, message: '请输入'  }]}]" placeholder="请输入"/>
                </a-form-item>
                <a-form-item label="选择银行">
                    <a-select v-decorator="['type',{ rules: [{ required: true, message: '请选择' }] }]" placeholder="请选择银行">
                        <a-select-option :value="value.id" v-for="(value,key) in typeList">
                            {{value.name}}
                        </a-select-option>
                    </a-select>
                </a-form-item>
                <a-form-item label="价格">
                    <a-input v-decorator="[`price`, { rules: [{ required: true, message: '请输入' }]}]" placeholder="请输入"/>
                </a-form-item>
                <a-form-item label="周期(天)">
                    <a-input v-decorator="[`day`, { rules: [{ required: true, message: '请输入' }]}]" placeholder="请输入"/>
                </a-form-item>
                <a-form-item label="内容">
                    <a-textarea v-decorator="[`content`, { rules: [{ required: true, message: '请输入' }]}]" placeholder="请输入" :rows="2"/>
                </a-form-item>
                <a-form-item label="收益率">
                    <a-input v-decorator="[`yield`, { rules: [{ required: true, message: '请输入' }]}]" placeholder="请输入"/>
                </a-form-item>
                <a-form-item label="状态">
                    <a-radio-group v-decorator="[`status`,{ rules: [{ required: true, message: '请选择' }]}]">
                        <a-radio value="0">进行中</a-radio>
                        <a-radio value="1">已结束</a-radio>
                    </a-radio-group>
                </a-form-item>
                <a-form-item label="排序">
                    <a-input v-decorator="[`sort`, { rules: [{ required: true, message: '请输入' }],initialValue:`0`}]" placeholder="请输入"/>
                </a-form-item>
            </a-form>
        </a-modal>
    </PageView>
</template>

<script type="text/jsx">
    import Bank from '../../api/Bank'
    import listMixin from '../mixin/listMixin'

    export default {
        name: 'productList',
        mixins: [listMixin],
        data () {
            return {
                columns: [
                    {
                        title: '名称',
                        dataIndex: 'name',
                        customRender: (v) => <a>{v}</a>
                    },
                    {
                        title: '价格',
                        dataIndex: 'price'
                    },
                    {
                        title: '周期 (天)',
                        dataIndex: 'day'
                    },
                    {
                        title: '内容',
                        dataIndex: 'content'
                    },
                    {
                        title: '收益率',
                        dataIndex: 'yield'
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
                                            this.showDialog(v)
                                        }}>
                                            <a-icon type="edit"/>
                                            修改
                                        </a-menu-item>
                                    </a-menu>
                                    <a-button>操作 <a-icon type="down"/></a-button>
                                </a-dropdown>
                            )
                        }
                    }
                ],
                visible: false,
                confirmLoading: false,
                form: this.$form.createForm(this),
                id: undefined,
                typeList:[]
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
                Bank.productList().then(res => {
                    this.data = res.data.map((value, key) => {
                        return { ...value, key }
                    })
                    this.loading = false
                })
            },
            showDialog (form) {
                this.id = form ? form.id : undefined
                this.visible = true
                this.form.resetFields()
                if (form) {
                    this.$nextTick(() => {
                        Object.keys(this.form.getFieldsValue()).forEach(v=>{
                            this.form.setFieldsValue({[v]: form[v]})
                        })
                    })
                }
            },
            start () {
                this.form.validateFields((err, values) => {
                    if (!err) {
                        this.confirmLoading = true
                        Bank.addProduct({ ...values, id: this.id }).then(res => {
                            this.confirmLoading = false
                            this.visible = false
                            this.getList()
                        }).catch(res => {
                            this.confirmLoading = false
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