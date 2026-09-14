<template>
    <PageView>
        <a-card>
            <div class="days-filter">
                <div class="days-filter-left">
                    <span class="days-filter-label">释放天数：</span>
                    <a-radio-group v-model="filterDays" buttonStyle="solid" @change="onDaysChange">
                        <a-radio-button :value="300">300天</a-radio-button>
                        <a-radio-button :value="600">600天</a-radio-button>
                        <a-radio-button :value="750">750天</a-radio-button>
                    </a-radio-group>
                </div>
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
                    <a-input v-model="desc" placeholder="请输入商品描述" />
                </a-form-item>
                <a-form-item label="商品金额" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input-number v-model="amount" :min="1" placeholder="请输入商品金额" style="width: 100%" />
                </a-form-item>
                <a-form-item label="日封顶" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input-number v-model="dailyCap" :min="0" placeholder="请输入日封顶" style="width: 100%" />
                </a-form-item>
                <a-form-item label="释放天数" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-select v-model="days" disabled style="width: 100%">
                        <a-select-option :value="300">300 天</a-select-option>
                        <a-select-option :value="600">600 天</a-select-option>
                        <a-select-option :value="750">750 天</a-select-option>
                    </a-select>
                </a-form-item>
                <a-form-item label="排序" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input-number v-model="sortOrder" :min="0" placeholder="数字越小越靠前" style="width: 100%" />
                </a-form-item>
                <a-form-item label="商品图片" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <div>
                        <img v-if="imageUrl" :src="imageUrl" class="goods-preview" @click="showImage(imageUrl)" />
                        <a-upload
                            name="file"
                            :multiple="false"
                            accept=".jpg,.jpeg,.png,.webp"
                            :showUploadList="false"
                            :customRequest="customRequest"
                        >
                            <a-button>
                                <a-icon type="upload" />{{ imageFile ? imageFile.name : (imageUrl ? '重新上传图片' : '上传图片') }}
                            </a-button>
                        </a-upload>
                    </div>
                </a-form-item>
                <a-form-item label="上架" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-switch v-model="enabled" />
                </a-form-item>
            </a-form>
        </a-modal>

        <a-modal title="商品图片" :visible="previewVisible" :footer="null" centered width="720px" @cancel="previewVisible = false">
            <img :src="previewUrl" class="goods-preview-large" />
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

const isOnSale = (row) => row && (row.on_sale === 1 || row.on_sale === '1')

export default {
    name: 'dealList3',
    mixins: [listMixin],
    data() {
        return {
            isShowJf: false,
            filterDays: 300,
            editId: 0,
            name: '',
            desc: '',
            amount: undefined,
            dailyCap: undefined,
            days: 300,
            sortOrder: 0,
            enabled: true,
            imageFile: null,
            imageUrl: '',
            previewVisible: false,
            previewUrl: '',
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
                    title: '图片',
                    dataIndex: 'image',
                    customRender: (v) => {
                        if (!v) return '-'
                        return <img src={v} style="display:block;height:40px;cursor:pointer;" onClick={() => this.showImage(v)} />
                    },
                },
                {
                    title: '名称',
                    dataIndex: 'name',
                },
                {
                    title: '描述',
                    dataIndex: 'desc',
                },
                {
                    title: '金额',
                    dataIndex: 'amount',
                    customRender: (v) => trimAmount(v),
                },
                {
                    title: '日封顶',
                    dataIndex: 'daily_cap',
                    customRender: (v) => trimAmount(v),
                },
                {
                    title: '排序',
                    dataIndex: 'sort',
                },
                {
                    title: '状态',
                    dataIndex: 'on_sale',
                    width: 120,
                    customRender: (v, row) => {
                        return (
                            <div style="display:flex;align-items:center;white-space:nowrap;">
                                <span style="margin-right:8px;color:rgba(0,0,0,.65);">上架</span>
                                <a-switch checked={isOnSale(row)} onChange={() => this.toggleStatus(row)} />
                            </div>
                        )
                    },
                },
                {
                    title: '操作',
                    key: 'action',
                    fixed: 'right',
                    width: 200,
                    customRender: (v) => {
                        return (
                            <div>
                                <a-button type="primary" icon="edit" onClick={() => this.openEdit(v)}>编辑</a-button>
                                <a-button type="danger" icon="delete" style="margin-left:8px;" onClick={() => this.remove(v)}>删除</a-button>
                            </div>
                        )
                    },
                }
            ],
            searchData: {},
        }
    },
    methods: {
        onDaysChange() {
            this.current = 1
            this.getList()
        },
        showImage(v) {
            if (!v) return
            this.previewUrl = v
            this.previewVisible = true
        },
        resetForm() {
            this.editId = 0
            this.name = ''
            this.desc = ''
            this.amount = undefined
            this.dailyCap = 0
            this.days = this.filterDays
            this.sortOrder = 0
            this.enabled = true
            this.imageFile = null
            this.imageUrl = ''
        },
        openCreate() {
            this.resetForm()
            this.isShowJf = true
        },
        openEdit(row) {
            this.editId = row.id
            this.name = row.name || ''
            this.desc = row.desc || ''
            this.amount = Number(row.amount)
            this.dailyCap = Number(row.daily_cap || 0)
            this.days = Number(row.days || this.filterDays)
            this.sortOrder = Number(row.sort || 0)
            this.enabled = isOnSale(row)
            this.imageFile = null
            this.imageUrl = row.image || ''
            this.isShowJf = true
        },
        customRequest(info) {
            const file = info.file
            const okType = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'].includes(file.type)
            if (!okType) {
                this.$message.info('图片格式不支持')
                if (info.onError) info.onError()
                return
            }
            if (file.size > 5 * 1024 * 1024) {
                this.$message.info('图片不能超过5MB')
                if (info.onError) info.onError()
                return
            }
            this.imageFile = file
            this.imageUrl = URL.createObjectURL(file)
            if (info.onSuccess) info.onSuccess()
        },
        uploadImage() {
            if (!this.imageFile) return Promise.resolve(this.editId ? '' : this.imageUrl)
            const formData = new FormData()
            formData.append('file', this.imageFile)
            formData.append('days', this.days)
            return Gai.web3_goods_image_upload(formData).then((res) => {
                if (res.status && res.status !== 'ok') {
                    return Promise.reject(new Error(res.status))
                }
                if (!res.url) {
                    return Promise.reject(new Error('图片上传失败'))
                }
                return res.url
            })
        },
        payload(image) {
            const data = {
                days: this.days,
                name: this.name,
                desc: this.desc,
                amount: this.amount,
                daily_cap: this.dailyCap,
                sort: this.sortOrder,
                on_sale: this.enabled ? 1 : 0,
            }
            if (this.editId) data.id = this.editId
            if (image) data.image = image
            return data
        },
        handleSave() {
            if (!this.name) return this.$message.info('请输入商品名称')
            if (!this.desc) return this.$message.info('请输入商品描述')
            if (!this.amount) return this.$message.info('请输入商品金额')
            if (![300, 600, 750].includes(Number(this.days))) return this.$message.info('请选择释放天数')
            this.confirmLoading = true
            this.uploadImage().then((image) => {
                const req = this.editId ? Gai.web3_goods_update(this.payload(image)) : Gai.web3_goods_create(this.payload(image))
                return req.then((res) => {
                    if (res.status && res.status !== 'ok') {
                        this.$message.error(res.status)
                        return
                    }
                    this.$message.success(this.editId ? '修改成功' : '新增成功')
                    this.getList()
                    this.isShowJf = false
                })
            }).catch((err) => {
                this.$message.error((err && err.message) || (this.editId ? '修改失败' : '新增失败'))
            }).finally(() => {
                this.confirmLoading = false
            })
        },
        toggleStatus(row) {
            const on = !isOnSale(row)
            this.$confirm({
                title: on ? '上架提示' : '下架提示',
                content: on ? '确定上架该商品到 Web3 商城？' : '确定从 Web3 商城下架该商品？',
                centered: true,
                onOk: () => {
                    return Gai.web3_goods_status({
                        id: row.id,
                        days: this.filterDays,
                        on_sale: on ? 1 : 0,
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
                    return Gai.web3_goods_delete({
                        id: row.id,
                        days: this.filterDays,
                    }).then((res) => {
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
            Gai.web3_goods_list({
                page: this.current,
                page_size: this.pageSize,
                days: this.filterDays,
            }).then((res) => {
                this.data = (res.list || []).map((value, key) => {
                    return { ...value, key }
                })
                this.total = parseInt(res.count || 0)
                this.loading = false
            }).catch(() => {
                this.loading = false
            })
        },
    },
}
</script>

<style scoped lang="less">
.days-filter {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
}
.days-filter-left {
    display: flex;
    align-items: center;
}
.days-filter-label {
    margin-right: 12px;
    color: rgba(0, 0, 0, 0.65);
    white-space: nowrap;
}
.goods-preview {
    display: block;
    height: 64px;
    margin-bottom: 8px;
    object-fit: contain;
    cursor: pointer;
}
.goods-preview-large {
    display: block;
    width: 100%;
    max-height: 70vh;
    object-fit: contain;
}
</style>
