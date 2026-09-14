<template>
    <PageView>
        <a-card title="Web3商品列表">
            <div slot="extra">
                <a-button type="primary" @click="openCreate">新增商品</a-button>
            </div>
            <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="{ total, pageSize, current }"
                @change="changePagination" bordered :scroll="{ x: true }">
            </a-table>
        </a-card>

        <a-modal :title="editId ? '修改商品' : '新增商品'" :visible="isShowJf" @ok="handleSave" :confirmLoading="confirmLoading"
            centered :closable="false" @cancel="isShowJf = false" :maskClosable="false" width="580px">
            <a-form style="margin-top: 20px">
                <a-form-item label="商品名称" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input v-model="name" placeholder="请输入商品名称" />
                </a-form-item>
                <a-form-item label="商品描述" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input v-model="goods" placeholder="请输入商品描述" />
                </a-form-item>
                <a-form-item label="商品金额" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input-number v-model="amount" :min="1" placeholder="请输入商品金额" style="width: 100%" />
                </a-form-item>
                <a-form-item label="日封顶" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input-number v-model="dailyCap" :min="0" placeholder="请输入日封顶" style="width: 100%" />
                </a-form-item>
                <a-form-item label="释放天数" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-select v-model="days" placeholder="请选择释放天数" style="width: 100%">
                        <a-select-option :value="300">300 天</a-select-option>
                        <a-select-option :value="600">600 天</a-select-option>
                        <a-select-option :value="750">750 天</a-select-option>
                    </a-select>
                </a-form-item>
                <a-form-item label="排序" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input-number v-model="sortOrder" :min="0" placeholder="数字越小越靠前" style="width: 100%" />
                </a-form-item>
                <a-form-item label="上架" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-switch v-model="enabled" />
                </a-form-item>
            </a-form>
        </a-modal>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'

const trimAmount = (v) => {
    if (v == null || v === '') return '0'
    const s = String(v).trim()
    if (!/^-?\d+(\.\d+)?$/.test(s)) return String(v)
    return s.replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '')
}

export default {
    name: 'dealList3',
    mixins: [listMixin],
    data() {
        return {
            isShowJf: false,
            editId: 0,
            name: '',
            goods: '',
            amount: undefined,
            dailyCap: undefined,
            days: 300,
            sortOrder: 0,
            enabled: true,
            labelCol: {
                xs: { span: 24 },
                sm: { span: 6 },
            },
            wrapperCol: {
                xs: { span: 24 },
                sm: { span: 18 },
            },
            confirmLoading: false,
            columns: [
                {
                    title: 'ID',
                    dataIndex: 'id',
                },
                {
                    title: '名称',
                    dataIndex: 'name',
                },
                {
                    title: '描述',
                    dataIndex: 'one',
                },
                {
                    title: '金额',
                    dataIndex: 'amount',
                    customRender: (v) => trimAmount(v),
                },
                {
                    title: '释放天数',
                    dataIndex: 'days',
                },
                {
                    title: '日封顶',
                    dataIndex: 'dailyCap',
                    customRender: (v) => trimAmount(v),
                },
                {
                    title: '排序',
                    dataIndex: 'sort_order',
                },
                {
                    title: '状态',
                    dataIndex: 'status',
                    customRender: (v) => {
                        if (v === '1' || v === true) return '上架中'
                        return '下架中'
                    }
                },
                {
                    title: '操作',
                    key: 'action',
                    fixed: 'right',
                    width: 220,
                    customRender: (v) => {
                        return (
                            <div>
                                <a-button-group>
                                    <a-button type="primary" onClick={() => this.openEdit(v)}>修改</a-button>
                                    <a-button onClick={() => this.toggleStatus(v)}>{v.status === '1' || v.enabled ? '下架' : '上架'}</a-button>
                                    <a-button type="danger" onClick={() => this.remove(v)}>删除</a-button>
                                </a-button-group>
                            </div>
                        )
                    },
                }
            ],
            searchData: {},
        }
    },
    methods: {
        payload() {
            return {
                id: this.editId || undefined,
                name: this.name,
                title: this.name,
                goods: this.goods || this.name,
                one: this.goods || this.name,
                amount: this.amount,
                daily_cap: this.dailyCap,
                dailyCap: this.dailyCap,
                days: this.days,
                release_days: this.days,
                sort_order: this.sortOrder,
                enabled: this.enabled ? 1 : 0,
                status: this.enabled ? 1 : 0,
            }
        },
        openCreate() {
            this.editId = 0
            this.name = ''
            this.goods = ''
            this.amount = undefined
            this.dailyCap = 0
            this.days = 300
            this.sortOrder = 0
            this.enabled = true
            this.isShowJf = true
        },
        openEdit(row) {
            this.editId = row.id
            this.name = row.name || row.title || ''
            this.goods = row.one || row.goods || ''
            this.amount = Number(row.amount)
            this.dailyCap = Number(row.dailyCap || row.daily_cap || 0)
            this.days = Number(row.days || row.release_days || 300)
            this.sortOrder = Number(row.sort_order || row.sortOrder || 0)
            this.enabled = row.status === '1' || row.enabled === true
            this.isShowJf = true
        },
        handleSave() {
            if (!this.name) return this.$message.info('请输入商品名称')
            if (!this.amount) return this.$message.info('请输入商品金额')
            if (![300, 600, 750].includes(Number(this.days))) return this.$message.info('请选择释放天数')
            this.confirmLoading = true
            const req = this.editId ? Gai.package_update(this.payload()) : Gai.package_create(this.payload())
            req.then(res => {
                if (res.status && res.status !== 'ok') {
                    this.$message.error(res.status)
                    return
                }
                this.$message.success(this.editId ? '修改成功' : '新增成功')
                this.getList()
                this.isShowJf = false
            }).catch(() => {
                this.$message.error(this.editId ? '修改失败' : '新增失败')
            }).finally(() => {
                this.confirmLoading = false
            })
        },
        toggleStatus(row) {
            const on = !(row.status === '1' || row.enabled === true)
            this.$confirm({
                title: on ? '上架提示' : '下架提示',
                content: on ? '确定上架该商品到 Web3 商城？' : '确定从 Web3 商城下架该商品？',
                centered: true,
                onOk: () => {
                    return Gai.package_update({
                        id: row.id,
                        enabled: on ? 1 : 0,
                        status: on ? 1 : 0,
                    }).then((res) => {
                        if (res.status && res.status !== 'ok') {
                            this.$message.error(res.status)
                            return Promise.reject()
                        }
                        this.getList()
                    })
                },
            })
        },
        remove(row) {
            this.$confirm({
                title: '删除商品',
                content: '已有订单的商品不能删除，请先下架。确定删除？',
                centered: true,
                onOk: () => {
                    return Gai.package_delete({ id: row.id }).then((res) => {
                        if (res.status && res.status !== 'ok') {
                            this.$message.error(res.status)
                            return Promise.reject()
                        }
                        this.$message.success('已删除')
                        this.getList()
                    })
                },
            })
        },
        getList() {
            this.loading = true
            Gai.trade_list_3({
                page: this.current,
                ...this.searchData
            }).then((res) => {
                this.data = (res.goods || []).map((value, key) => {
                    return { ...value, key }
                })
                this.loading = false
                this.total = parseInt(res.count || 0)
            })
        },
    },
}
</script>

<style scoped lang="less">
.inputGroup {
    >div {
        margin-bottom: 20px;
    }
}
</style>
