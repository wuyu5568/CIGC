<template>
    <PageView>
        <a-card title="列表">
            <!-- <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="账户" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-button-group>
                        <a-button type="primary" :loading="loading" @click="getListTwo">确定筛选</a-button>
                    </a-button-group>
                </a-col>
            </a-row> -->
            <a-button type="primary" slot="extra" @click="isShowJf = true">新增商品</a-button>
            <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="{ total, pageSize, current }"
                @change="changePagination" bordered :scroll="{ x: true }">
            </a-table>
        </a-card>

        <a-modal title="新增商品" :visible="isShowJf" @ok="handleCreate" :confirmLoading="confirmLoading"
            centered :closable="false" @cancel="isShowJf = false" :maskClosable="false" width="580px">
            <a-form :form="form" style="margin-top: 20px">
                <a-form-item label="商品名称" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input v-model="name" placeholder="请输入商品名称" />
                </a-form-item>
                <a-form-item label="商品简介" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-input v-model="detail" placeholder="请输入商品简介" />
                </a-form-item>
                <a-form-item label="商品详情" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-textarea v-model:value="three" placeholder="请输入商品详情" :rows="4" />
                </a-form-item>
                <a-form-item label="商品金额" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <!-- <a-select v-model.number="amount" placeholder="请选择商品金额">
                        <a-select-option v-for="item in amountOptions" :value="item.value">{{ item.label }}</a-select-option>
                    </a-select> -->
                    <a-input-number v-model="amount" :min="1" prefix="￥" placeholder="请输入商品金额" style="width: 100%" />
                </a-form-item>
                <a-form-item label="商品封面" :label-col="labelCol" :wrapper-col="wrapperCol">
                    <a-upload
                        name="file"
                        :multiple="false"
                        :customRequest="customRequest"
                        :showUploadList="false"
                        @change="handleChange"
                    >
                        <a-button>
                            <a-icon type="upload" />{{file ? file.name : '点击上传封面图'}}
                        </a-button>
                    </a-upload>
                </a-form-item>
            </a-form>
        </a-modal>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'
export default {
    name: 'dealList',
    mixins: [listMixin],
    data() {
        return {
            form: null,
            isShowJf: false,
            name: '',
            detail: '',
            three: '',
            file: null,
            amount: undefined,
            amountOptions: [
                { label: '100', value: 100 },
                { label: '300', value: 300 },
                { label: '500', value: 500 },
                { label: '1000', value: 1000 },
                { label: '5000', value: 5000 },
                { label: '10000', value: 10000 },
                { label: '15000', value: 15000 },
                { label: '30000', value: 30000 },
                { label: '50000', value: 50000 },
                { label: '100000', value: 100000 },
                { label: '150000', value: 150000 },
            ],
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
                    title: '商品图片',
                    dataIndex: 'two',
                    customRender: (v) => {
                        return <img src={`/images/${v}`} style="width: 100px; height: 100px;" />
                    }
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
                    title: '详情',
                    dataIndex: 'three',
                },
                {
                    title: '金额',
                    dataIndex: 'amount',
                },
                {
                    title: '状态',
                    dataIndex: 'status',
                    customRender: (v) => {
                        if (v === '1') return '上架中'
                        if (v === '0') return '下架中'
                    }
                },
                {
                    title: '操作',
                    key: 'action',
                    fixed: 'right',
                    width: 110,
                    customRender: (v) => {
                        return (
                            <div>
                                <a-button-group>
                                    <a-button type="primary" onClick={() => {
                                        this.handleState(v.id, v.status === '1' ? '0' : '1');
                                    }}>{v.status === '1' ? '下架' : '上架'}</a-button>
                                </a-button-group>
                            </div>
                        )
                    },
                }
            ],
            searchData: {
                address: '',
            },
        }
    },
    mounted() {
    },
    methods: {
        handleChange(file) {
            // this.file = file
        },
        customRequest(files) {
            this.file = files.file
        },
        handleCreate() {
            console.log(this.name, this.detail, this.three, this.amount)
            if (!this.name) return this.$message.info('请输入商品名称');
            if (!this.detail) return this.$message.info('请输入商品简介');
            if (!this.three) return this.$message.info('请上传商品详情');
            if (!this.amount) return this.$message.info('请输入商品金额');
            if (!this.file) return this.$message.info('请上传商品封面');

            const formData = new FormData();
            formData.append('name', this.name);
            formData.append('one', this.detail);
            formData.append('amount', this.amount);
            formData.append('three', this.three);
            formData.append('file', this.file);
            Gai.product_cover_upload_2(formData).then(res => {
                this.getList()
                this.isShowJf = false
                this.name = ''
                this.detail = ''
                this.amount = null
                this.file = null
            }).catch(res => {
                console.log(res)
            })
        },
        handleState(user_id, status) {
            let text = status === "1" ? `上架` : `下架`;
            this.$confirm({
                title: `${text}提示`,
                content: `确定要${text}商品吗？`,
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.product_status_2({ id: user_id, status }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        getList() {
            this.loading = true
            Gai.trade_list_2({
                page: this.current,
                ...this.searchData
            }).then((res) => {
                this.data = res.goods
                this.loading = false
                this.total = parseInt(res.count)
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