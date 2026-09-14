<template>
    <PageView>
        <a-card title="列表">
            <a-button type="primary" slot="extra" @click="changeConfig()">新增管理员</a-button>
            <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="false"
                @change="changePagination" bordered :scroll="{ x: true }">
            </a-table>
        </a-card>
        <!--新增管理员-->
        <a-modal :title="id ? `修改密码` : `新增管理员`" :visible="isShowJf" @ok="editAdmin" :confirmLoading="confirmLoading"
            centered :closable="false" @cancel="isShowJf = false" :maskClosable="false" width="380px">
            <a-form :form="form" style="margin-top: 20px">
                <a-form-item label="账号" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input :disabled="!!id"
                        v-decorator="['account', { rules: [{ required: true, message: '请输入账号' }] }]"
                        placeholder="请输入账号" />
                </a-form-item>
                <a-form-item label="密码" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input v-decorator="['password', { rules: [{ required: true, message: '请输入密码' }] }]"
                        placeholder="请输入密码" />
                </a-form-item>
            </a-form>
        </a-modal>
        <!-- 用户权限 -->
        <a-modal title="权限" :visible="showAuth" centered :closable="false" :maskClosable="false" width="380px">
            <a-spin :spinning="spinning">
                <div class="auth_list">
                    <div v-for="v in allList">
                        <div class="name">
                            {{ v.name }}
                        </div>
                        <a-button v-if="userList.includes(v.path)" type="danger"
                            @click="changeStatus(v.id, false)">关闭</a-button>
                        <a-button v-else type="primary" @click="changeStatus(v.id, true)">开通</a-button>
                    </div>
                </div>
            </a-spin>
            <template slot="footer">
                <a-button @click="showAuth = false">关闭</a-button>
            </template>
        </a-modal>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin';

export default {
    name: 'admin',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: '管理员ID',
                    dataIndex: 'id',
                    customRender: (v) => <a>{v}</a>
                },
                {
                    title: '管理员账号',
                    dataIndex: 'account',
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
                                    type="primary"
                                    onClick={() => {
                                        this.changeConfig(v)
                                    }}>
                                    修改密码
                                </a-button>
                                <a-button
                                    type="primary"
                                    onClick={() => {
                                        this.id = v.id;
                                        this.userList = [];
                                        this.spinning = true;
                                        this.user_auth_list();
                                        this.showAuth = true;
                                    }}>
                                    权限
                                </a-button>
                            </a-button-group>
                        )
                    }
                }
            ],
            allList: [],
            userList: [],
            spinning: false,
            form: this.$form.createForm(this),
            confirmLoading: false,
            isShowJf: false,
            showAuth: false,
            labelCol: {
                xs: { span: 24 },
                sm: { span: 6 },
            },
            wrapperCol: {
                xs: { span: 24 },
                sm: { span: 18 },
            },
            id: undefined,
        }
    },
    created() {
        this.auth_list();
    },
    methods: {
        auth_list() {
            Gai.auth_list().then(res => {
                this.allList = res.auth;
            })
        },
        getList() {
            this.loading = true
            Gai.admin_list().then(res => {
                this.data = res.account.map((value, key) => {
                    return { ...value, key }
                })
                this.loading = false
            })
        },
        changeConfig(v) {
            this.id = v ? v.id : undefined;
            this.isShowJf = true;
            this.form.resetFields();
            if (v) {
                this.$nextTick(() => {
                    this.form.setFieldsValue({ "account": v.account })
                })
            }
        },
        editAdmin() {
            this.form.validateFields((err, values) => {
                if (!err) {
                    this.confirmLoading = false
                    Gai[this.id ? `change_password` : `create_account`](values).then(res => {
                        this.isShowJf = false
                        this.confirmLoading = false
                        this.getList()
                    }).catch(res => {
                        this.confirmLoading = false
                    })
                }
            });
        },
        user_auth_list() {
            Gai.user_auth_list({
                admin_id: this.id,
            }).then(res => {
                this.userList = res.auth.map(v => v.path);
                this.spinning = false;
            })
        },
        /* 修改用户权限 */
        changeStatus(auth_id, bool) {
            this.spinning = true;
            Gai[bool ? `auth_create` : `auth_delete`]({ auth_id, admin_id: this.id, }).then(res => {
                this.user_auth_list();
            }).catch(res => {
                this.spinning = false;
            })
        }
    }
}
</script>

<style scoped lang="less">
.inputGroup {
    >div {
        margin-bottom: 20px;
    }
}

.auth_list {
    >div {
        padding: 10px 0px;
        display: flex;
        align-items: center;
        justify-content: space-between;
    }
}
</style>