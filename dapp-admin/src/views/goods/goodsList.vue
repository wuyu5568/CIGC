<template>
    <PageView>
        <a-card title="列表">
            <a-button type="primary" slot="extra" @click="showDialog()">新增商品</a-button>
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
        <a-modal
            :title="form.id ? `修改` : `新增`"
            :visible="isShow"
            centered
            :closable="false"
            :maskClosable="false"
            width="400px"
        >
            <a class="radioTitle">商品名称</a>
            <a-input v-model="form.name" placeholder="商品名称" />
            <a class="radioTitle">价格区间(最小)</a>
            <a-input v-model="form.money_begin" placeholder="价格区间" />
            <a class="radioTitle">价格区间(最大)</a>
            <a-input v-model="form.money_over" placeholder="价格区间" />
            <a class="radioTitle">合约期</a>
            <a-input v-model="form.day" placeholder="合约期" />
            <a class="radioTitle">利润</a>
            <a-input v-model="form.profit" placeholder="利润" />
            <a class="radioTitle">图片</a>
            <img class="goodsPic" v-if="form.image" :src="imgUrl + form.image" />
            <div class="btnCon">
                <input @change="handleChange" type="file" accept="image/*" />
                <a-button-group>
                    <a-button type="primary">选择图片</a-button>
                </a-button-group>
            </div>
            <template slot="footer">
                <a-button-group>
                    <a-button type="primary" :loading="btnloading" @click="addProject()">确定</a-button>
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
    name: 'goodsList',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: '图片',
                    dataIndex: 'pic',
                    customRender: (v) => <img style="display:block;height:40px;cursor:pointer;" src={projectUrl+v} />,
                },
                {
                    title: '商品名称',
                    dataIndex: 'name',
                },
                {
                    title: '利润',
                    dataIndex: 'profit',
                },
                {
                    title: '合约期',
                    dataIndex: 'day',
                },
                {
                    title: '价格区间',
                    customRender: (v) => `${v.money_begin} - ${v.money_over}`,
                },
                {
                        title: '状态',
                        dataIndex: 'status',
                        customRender: (v) => {
                            let status = {
                                "-1": { name: '已下架', color: 'error' },
                                "1": { name: '已上架', color: 'success' },
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
                                    <a-menu-item
                                        onClick={() => {
                                            this.showDialog(v)
                                        }}
                                    >
                                        修改信息
                                    </a-menu-item>
                                    <a-menu-item
                                        onClick={() => {
                                            this.updateSaleStatus(v)
                                        }}
                                    >
                                        {v.status === "1"?`下架`:`上架`}
                                    </a-menu-item>
                                </a-menu>
                                <a-button>
                                    操作 <a-icon type="down" />
                                </a-button>
                            </a-dropdown>
                        )
                    },
                },
            ],
            isShow: false,
            form: {
                id: undefined,
                name: undefined,
                money_begin: undefined,
                money_over: undefined,
                day: undefined,
                image: undefined,
                profit:undefined,
            },
            btnloading: false,
            imgUrl:projectUrl,
        }
    },
    methods: {
        getList() {
            this.loading = true
            Project.getAllProject({
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
        showDialog(v) {
            this.form.id = v ? v.id : undefined
            this.form.name = v ? v.name : undefined
            this.form.money_begin = v ? v.money_begin : undefined
            this.form.money_over = v ? v.money_over : undefined
            this.form.day = v ? v.day : undefined
            this.form.image = v ? v.pic : undefined
            this.form.profit = v ? v.profit :undefined
            this.isShow = true
        },
        handleChange(e) {
            let file = e.target.files[0]
            let formData = new FormData()
            formData.append('pic', file)
            Project.updatePic(formData).then((res) => {
                this.form.image = res.data.url
            })
        },
        addProject() {
            this.btnloading = true
            Project.addProject(this.form)
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
        updateSaleStatus(v){
            this.$confirm({
                title: v.status === "1"?`下架`:`上架`,
                content: `确定要${v.status === "1"?`下架`:`上架`}此商品吗?`,
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Project.updateSaleStatus({ id:v.id }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        }
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
.goodsPic {
    display: block;
    width: 200px;
}
.btnCon {
    margin-top: 20px;
    position: relative;
    overflow: hidden;
    display: inline-block;
    input {
        display: block;
        position: absolute;
        z-index: 1;
        width: 100%;
        height: 100%;
        background: red;
        opacity: 0;
    }
}
</style>