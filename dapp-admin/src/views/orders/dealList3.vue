<template>
    <PageView>
        <a-card>
            <div class="goods-toolbar">
                <a-button type="primary" @click="openCreate">新增商品</a-button>
            </div>
            <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="{ total, pageSize, current }"
                @change="changePagination" bordered :scroll="{ x: true }">
            </a-table>
        </a-card>

        <a-modal :title="editId ? '修改商品' : '新增商品'" :visible="isShowJf" @ok="handleSave" :confirmLoading="confirmLoading"
            centered :closable="false" @cancel="isShowJf = false" :maskClosable="false" width="960px" destroyOnClose
            :bodyStyle="{ maxHeight: '72vh', overflow: 'auto' }">
            <a-form style="margin-top: 20px">
                <a-form-item label="名称" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input v-model="name" placeholder="请输入商品名称" />
                </a-form-item>
                <a-form-item label="描述" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-textarea v-model="desc" :rows="2" placeholder="请输入商品描述" />
                </a-form-item>
                <a-form-item label="主图" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <div>
                        <div v-if="imageUrl" class="goods-cover">
                            <img :src="imageUrl" class="goods-preview" @click="showImage(imageUrl)" />
                            <a-button type="danger" ghost size="small" @click="clearImage">删除主图</a-button>
                        </div>
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
                <a-form-item label="详情" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-checkbox :checked="detailEnabled" @change="onDetailEnabled">填写详情</a-checkbox>
                    <div v-if="detailEnabled" class="detail-editor">
                        <tinymceForm
                            editor-id="web3-goods-detail"
                            :height="360"
                            :value="detail"
                            :upload-handler="onDetailImageUpload"
                            @input="onDetailInput"
                        />
                    </div>
                </a-form-item>
                <a-form-item label="单价" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input-number v-model="amount" :min="1" placeholder="请输入单价" style="width: 100%" />
                    <div class="cap-hint">对应日封顶 {{ capHint }} USDT（按结算金额自动套档，不可配置）</div>
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
import tinymceForm from '../../components/tinymceForm/tinymceForm'

const trimAmount = (v) => {
    if (v == null || v === '') return '0'
    const s = String(v).trim()
    if (!/^-?\d+(\.\d+)?$/.test(s)) return String(v)
    return s.replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '')
}

const isOnSale = (row) => row && (row.on_sale === 1 || row.on_sale === '1')

const hasDetailHtml = (s) => htmlToPlain(s).length > 0

const htmlToPlain = (s) => {
    return String(s || '')
        .replace(/<br\s*\/?>/gi, '\n')
        .replace(/<\/p>/gi, '\n')
        .replace(/<[^>]+>/g, '')
        .replace(/&nbsp;/gi, ' ')
        .replace(/&lt;/g, '<')
        .replace(/&gt;/g, '>')
        .replace(/&amp;/g, '&')
        .replace(/\n{3,}/g, '\n\n')
        .trim()
}

const capForAmount = (amount) => {
    const a = Number(amount)
    if (!Number.isFinite(a) || a <= 0) return '0'
    if (a < 3000) return '600'
    if (a < 6000) return '1800'
    if (a < 12000) return '4000'
    if (a < 24000) return '16000'
    if (a < 36000) return '24000'
    if (a < 50000) return '30000'
    if (a < 70000) return '42000'
    if (a < 100000) return '60000'
    return '100000'
}

export default {
    name: 'dealList3',
    mixins: [listMixin],
    components: { tinymceForm },
    data() {
        return {
            isShowJf: false,
            editId: 0,
            name: '',
            desc: '',
            detail: '',
            detailEnabled: false,
            amount: undefined,
            enabled: true,
            imageFile: null,
            imageUrl: '',
            imageCleared: false,
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
                    title: '主图',
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
                    title: '单价',
                    dataIndex: 'amount',
                    customRender: (v) => trimAmount(v),
                },
                {
                    title: '日封顶',
                    dataIndex: 'daily_cap',
                    customRender: (v, row) => trimAmount(v || capForAmount(row && row.amount)),
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
                    customRender: (v, row) => {
                        return (
                            <div>
                                <a-button type="primary" icon="edit" onClick={() => this.openEdit(row)}>编辑</a-button>
                                <a-button type="danger" icon="delete" style="margin-left:8px;" onClick={() => this.remove(row)}>删除</a-button>
                            </div>
                        )
                    },
                }
            ],
            searchData: {},
        }
    },
    computed: {
        capHint() {
            return capForAmount(this.amount)
        },
    },
    methods: {
        showImage(v) {
            if (!v) return
            this.previewUrl = v
            this.previewVisible = true
        },
        resetForm() {
            this.editId = 0
            this.name = ''
            this.desc = ''
            this.detail = ''
            this.detailEnabled = false
            this.amount = undefined
            this.enabled = true
            this.imageFile = null
            this.imageUrl = ''
            this.imageCleared = false
        },
        openCreate() {
            this.resetForm()
            this.isShowJf = true
        },
        openEdit(row) {
            this.editId = row.id
            this.name = row.name || ''
            this.desc = row.desc || ''
            this.detail = ''
            this.detailEnabled = !!(row.has_detail === true || row.has_detail === 1 || row.has_detail === '1')
            this.amount = Number(row.amount)
            this.enabled = isOnSale(row)
            this.imageFile = null
            this.imageUrl = row.image || ''
            this.imageCleared = false
            this.isShowJf = true
            Gai.web3_goods_detail({ id: row.id }).then((res) => {
                if (res.status && res.status !== 'ok') return
                const item = res.item || {}
                if (item.name != null) this.name = item.name
                if (item.desc != null) this.desc = item.desc
                this.detail = item.detail != null ? String(item.detail) : ''
                this.detailEnabled = hasDetailHtml(this.detail) || item.has_detail === true || item.has_detail === 1 || item.has_detail === '1'
                if (item.amount != null) this.amount = Number(item.amount)
                this.enabled = isOnSale(item)
                if (item.image != null) this.imageUrl = item.image || ''
            })
        },
        clearImage() {
            this.imageFile = null
            this.imageUrl = ''
            this.imageCleared = true
        },
        onDetailEnabled(e) {
            this.detailEnabled = !!(e && e.target && e.target.checked)
        },
        onDetailInput(v) {
            this.detail = v == null ? '' : String(v)
        },
        onDetailImageUpload(blobInfo, success, failure) {
            const blob = blobInfo && blobInfo.blob ? blobInfo.blob() : null
            if (!blob) {
                failure('请上传图片')
                return
            }
            if (blob.size > 5 * 1024 * 1024) {
                failure('图片不能超过5MB')
                return
            }
            const formData = new FormData()
            formData.append('file', blob, (blobInfo && blobInfo.filename && blobInfo.filename()) || 'image.png')
            Gai.web3_goods_image_upload(formData).then((res) => {
                if (res && res.url) {
                    success(res.url)
                    return
                }
                failure((res && res.status) || '图片上传失败')
            }).catch(() => {
                failure('图片上传失败')
            })
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
            this.imageCleared = false
            if (info.onSuccess) info.onSuccess()
        },
        uploadImage() {
            if (!this.imageFile) return Promise.resolve(this.editId ? '' : this.imageUrl)
            const formData = new FormData()
            formData.append('file', this.imageFile)
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
                name: this.name,
                desc: this.desc,
                detail: this.detailEnabled ? this.detail : '',
                amount: this.amount,
                on_sale: this.enabled ? 1 : 0,
            }
            if (this.editId) data.id = this.editId
            if (this.imageCleared) data.image = ''
            else if (image) data.image = image
            return data
        },
        handleSave() {
            if (!this.amount) return this.$message.info('请输入单价')
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
.goods-toolbar {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    margin-bottom: 16px;
}
.detail-editor {
    margin-top: 8px;
}
.cap-hint {
    margin-top: 6px;
    color: rgba(0, 0, 0, 0.45);
    font-size: 12px;
    line-height: 1.4;
}
.goods-cover {
    display: flex;
    align-items: flex-end;
    gap: 12px;
    margin-bottom: 8px;
}
.goods-preview {
    display: block;
    height: 64px;
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

<style lang="less">
.tox-tinymce-aux,
.tox-menu,
.tox-dialog-wrap {
    z-index: 4000 !important;
}
</style>
