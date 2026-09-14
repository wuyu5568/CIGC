<template>
    <PageView>
        <a-card title="列表">
            <a-button type="primary" slot="extra" @click="()=>{form.resetFields();isShowJf = true}">新增管理员</a-button>
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.username" placeholder="请输入账号" @keyup.enter="getListTwo"/>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.status" style="width:100%" placeholder="请选择状态"
                              @change="getListTwo">
                        <a-select-option value="1">正常</a-select-option>
                        <a-select-option value="-1">冻结</a-select-option>
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
        <!--新增管理员-->
        <a-modal
            title="新增管理员"
            :visible="isShowJf"
            @ok="addAdmin"
            :confirmLoading="confirmLoading"
            centered
            :closable="false"
            @cancel="isShowJf = false"
            :maskClosable="false"
            width="380px"
        >
            <a-form :form="form" style="margin-top: 20px">
                <a-form-item label="用户名" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input v-decorator="['username', { rules: [{ required: true, message: '请输入用户名' }] }]" placeholder="请输入用户名"/>
                </a-form-item>
                <a-form-item label="密码" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input v-decorator="['password', { rules: [{ required: true, message: '请输入密码' }] }]" placeholder="请输入密码"/>
                </a-form-item>
                <a-form-item label="确认密码" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input v-decorator="['repassword', { rules: [{ required: true, message: '请再次输入密码' }] }]" placeholder="请再次输入密码"/>
                </a-form-item>
            </a-form>
        </a-modal>
    </PageView>
</template>

<script type="text/jsx">
    import Auth from '../../api/Auth'
    import listMixin from '../mixin/listMixin'

    export default {
        name: 'admin',
        mixins:[listMixin],
        data () {
            return {
                columns: [
                    {
                        title: '管理员ID',
                        dataIndex: 'admin_id',
                        customRender: (v) => <a>{v}</a>
                    },
                    {
                        title: '管理员账号',
                        dataIndex: 'username',
                        customRender:(v,r)=>{
                            return r.isEdit?(
                                <a-input placeholder="请输入管理员账号" defaultValue={r.username} onChange={(e)=>{r.username = e.target.value}}/>
                            ):(
                                r.username
                            )
                        }
                    },
                    {
                        title: '密码',
                        dataIndex: 'password',
                        customRender:(v,r)=>{
                            return r.isEdit?(
                                <a-input placeholder="管理员密码(不填写则不修改)" defaultValue={r.password} onChange={(e)=>{r.password = e.target.value}}/>
                            ):(
                                "******"
                            )
                        }
                    },
                    {
                        title: '状态',
                        dataIndex: 'status',
                        customRender:(v,r)=>{
                            let status = {
                                1: { name: '正常', color: 'success' },
                                "-1": { name: '冻结', color: 'error' },
                            }
                            return r.isEdit?(
                                <a-select defaultValue={r.status} style="width:100%" placeholder="请选择状态" onChange={(e)=>{r.status = e;}}>
                                    <a-select-option value="1">正常</a-select-option>
                                    <a-select-option value="-1">冻结</a-select-option>
                                </a-select>
                            ):(
                                <a-badge status={status[v].color} text={status[v].name}/>
                            )
                        }
                    },
                    {
                        title: '操作',
                        key: 'action',
                        fixed: 'right',
                        width: 110,
                        customRender: (v) => {
                            return (
                                <a-button-group>
                                    <a-button
                                        type={v.isEdit?`danger`:`primary`}
                                        onClick={()=>{
                                            v.isEdit?this.changeConfig(v):v.isEdit = true
                                        }}>
                                        {v.isEdit?`确定`:`编辑`}
                                    </a-button>
                                    {v.isEdit?(
                                        <a-button type="danger" onClick={()=>this.getList()}>取消编辑</a-button>
                                    ):""}

                                </a-button-group>
                            )
                        }
                    }
                ],
                searchData: {
                    username: '',
                    status: undefined,
                },
                form: this.$form.createForm(this),
                confirmLoading:false,
                isShowJf:false,
                labelCol:{
                    xs:{span:24},
                    sm:{span:6},
                },
                wrapperCol:{
                    xs:{span:24},
                    sm:{span:18},
                }
            }
        },
        methods: {
            getList () {
                this.loading = true
                Auth.getAllAdmin({
                    page: this.current,
                    num: this.pageSize,
                    ...this.searchData
                }).then(res => {
                    this.data = res.data.map((value, key) => {
                        return { ...value, key, isEdit:false,password:""}
                    })
                    this.loading = false
                    this.total = parseInt(res.count)
                })
            },
            changeConfig(v){
                this.$confirm({
                    title: `修改`,
                    content: `确定要修改此管理员吗?`,
                    centered: true,
                    onOk: () => {
                        let config = {admin_id:v.admin_id,username:v.username,status:v.status,password:v.password?v.password:undefined}
                        return new Promise((resolve, reject) => {
                            Auth.changeInfo(config).then(res => {
                                resolve()
                                this.getList()
                            }).catch(res => {
                                reject()
                            })
                        })
                    }
                })
            },
            addAdmin(){
                this.form.validateFields((err, values) => {
                    if (!err) {
                        this.confirmLoading = false
                        Auth.addAdminMember(values).then(res => {
                            this.isShowJf = false
                            this.confirmLoading = false
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