<template>
    <PageView>
        <a-card title="列表">
            <a-button type="primary" slot="extra" @click="showDialog()">添加挂卖单</a-button>
            <a-table
                :loading="loading"
                :columns="columns"
                :dataSource="data"
                :pagination="{ total, pageSize, showSizeChanger, current }"
                @change="changePagination"
                bordered
                :scroll="{ x: true }"
            >
            </a-table>
        </a-card>
        <!--修改金额-->
        <a-modal title="金额" :visible="isShow" centered :closable="false" :maskClosable="false" width="360px">
            <a class="radioTitle">类型</a>
            <a-select allowClear v-model="form.project_id" style="width: 100%" placeholder="请选择商品">
                <a-select-option :value="v.id" v-for="(v, k) in goodsList" :key="k">{{ v.name }}</a-select-option>
            </a-select>
            <a class="radioTitle">场次</a>
            <a-select allowClear v-model="form.type" style="width: 100%;" placeholder="请选择场次">
                <a-select-option :value="v.id" v-for="(v, k) in ccList" :key="k">{{ v.name }}</a-select-option>
            </a-select>
            <a class="radioTitle">底价</a>
            <a-input v-model="form.price" placeholder="请输入底价"/>
            <a class="radioTitle">利润</a>
            <a-input v-model="form.profit" placeholder="请输入利润"/>
            <a class="radioTitle">数量</a>
            <a-input v-model="form.num" placeholder="请输入数量"/>
            <template slot="footer">
                <a-button-group>
                    <a-button type="primary" :loading="btnloading" @click="addSaleOrder">确定</a-button>
                    <a-button type="default" @click="isShow = false">关闭</a-button>
                </a-button-group>
            </template>
        </a-modal>
    </PageView>
</template>

<script type="text/jsx">
import Project from '../../api/Project'
import listMixin from '../mixin/listMixin'
export default {
    name: 'addGmOrder',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: '用户',
                    dataIndex: 'username',
                    customRender: (v) => <a>{v}</a>,
                },
                {
                    title: '金额',
                    dataIndex: 'money',
                },
                {
                    title: '商品',
                    dataIndex: 'project_name',
                },
                {
                    title: '场次',
                    dataIndex: 'type_name',
                },
                {
                    title: '状态',
                    dataIndex: 'status',
                    customRender: (v) => {
                        return{
                            0:"待交易",
                            1:"交易中",
                            2:"交易成功",
                            "-1":"失效",
                            "-2":"已赔付"
                        }[v]
                    },
                },
                {
                    title: '时间',
                    dataIndex: 'add_time',
                    customRender: (v) => v?this.timeOne(v):"无",
                },
            ],
            isShow: false,
            btnloading: false,
            form: {
                uid: undefined,
                project_id: undefined,
                type: undefined,
                price:undefined,
                profit:undefined,
                num:undefined,
            },
            goodsList: [],
            ccList: [],
        }
    },
    activated(){
        this.form.uid = this.$route.query.uid;
        this.getType();
    },
    methods: {
        getList() {
            this.loading = true
            Project.getSaleList({
                page: this.current,
                num: this.pageSize,
            }).then((res) => {
                this.data = res.data.map((value, key) => {
                    return { ...value, key }
                })
                this.loading = false
                this.total = parseInt(res.count)
            })
        },
        getType() {
            // 商品
            Project.getAllProject({
                page: 1,
                num: 200,
            }).then((res) => {
                this.goodsList = res.data
            })
            // 场次
            Project.getProjectType().then((res) => {
                this.ccList = res.data
            })
        },
        showDialog() {
            this.form.project_id = undefined
            this.form.type = undefined
            this.form.price = undefined
            this.form.profit = undefined
            this.form.num = undefined
            this.isShow = true
        },
        addSaleOrder() {
            this.btnloading = true
            Project.addSaleOrder(this.form)
                .then((res) => {
                    this.isShow = false
                    this.btnloading = false
                    this.getList()
                })
                .catch((res) => {
                    this.isShow = false
                    this.btnloading = false
                })
        },
    },
}
</script>

<style scoped lang="less">
.inputGroup {
    > div {
        margin-bottom: 20px;
    }
}
.radioTitle {
    display: block;
    font-size: 18px;
    padding: 20px 0px;
}
.radioStyle {
    display: block;
    height: 30px;
    line-height: 30px;
    margin-left: 20px;
}
</style>