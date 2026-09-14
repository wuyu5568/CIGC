<template>
    <PageView>
        <a-card title="列表">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="账户" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-button-group>
                        <a-button type="primary" :loading="loading" @click="getListTwo">确定筛选</a-button>
                    </a-button-group>
                </a-col>
            </a-row>
            <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="{ total, pageSize, current }"
                @change="changePagination" bordered :scroll="{ x: true }">
            </a-table>
        </a-card>

        <a-modal title="手动补差价" :visible="diffVisible" :confirmLoading="diffLoading" @ok="submitDiff"
            @cancel="diffVisible = false">
            <a-form>
                <a-form-item label="本单usdt差值" :labelCol="{ span: 8 }" :wrapperCol="{ span: 14 }">
                    <a-input v-model="diffForm.amountOne" type="number" placeholder="请输入" />
                </a-form-item>
                <a-form-item label="本单ispay差值" :labelCol="{ span: 8 }" :wrapperCol="{ span: 14 }">
                    <a-input v-model="diffForm.amountTwo" type="number" placeholder="请输入" />
                </a-form-item>
                <a-form-item label="本单每日释放ispay数字" :labelCol="{ span: 8 }" :wrapperCol="{ span: 14 }">
                    <a-input v-model="diffForm.amountThree" type="number" placeholder="请输入" />
                </a-form-item>
            </a-form>
        </a-modal>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'
export default {
    name: 'ordersList',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: '时间',
                    dataIndex: 'createdAt',
                },
                {
                    title: '金额',
                    dataIndex: 'amount',
                },
                {
                    title: '地址',
                    dataIndex: 'address',
                },
                {
                    title: '操作',
                    key: 'action',
                    fixed: 'right',
                    width: 130,
                    customRender: (v) => {
                        return (
                            <a-button type="primary" size="small" onClick={() => {
                                this.openDiff(v);
                            }}>手动补差价</a-button>
                        )
                    },
                },
                // {
                //     title: '商品名称',
                //     dataIndex: 'one',
                // },
                // {
                //     title: '收件人',
                //     dataIndex: 'four',
                // },
                // {
                //     title: '收件人手机号',
                //     dataIndex: 'three',
                // },
                // {
                //     title: '收货地址',
                //     dataIndex: 'two',
                // },
                // {
                //     title: '操作',
                //     key: 'action',
                //     fixed: 'right',
                //     width: 110,
                //     customRender: (v) => {
                //         return (
                //             <div>
                //                 <a-button-group>
                //                     <a-button type="primary" onClick={() => {
                //                         this.subscription_delete(v.id);
                //                     }}>删除</a-button>
                //                 </a-button-group>
                //             </div>
                //         )
                //     },
                // },
            ],
            searchData: {
                address: '',
                reason: 'buy',
            },
            diffVisible: false,
            diffLoading: false,
            diffForm: {
                id: undefined,
                address: '',
                amountOne: '',
                amountTwo: '',
                amountThree: '',
            },
        }
    },
    methods: {
        // 打开补差价弹窗，回填该行的 id 与 address
        openDiff(record) {
            this.diffForm = {
                id: record.id,
                address: record.address,
                amountOne: '',
                amountTwo: '',
                amountThree: '',
            }
            this.diffVisible = true
        },
        // 提交补差价：id、address 原样传，三个金额转为 double
        submitDiff() {
            this.diffLoading = true
            Gai.buy_four_diff({
                id: this.diffForm.id,
                address: this.diffForm.address,
                amountOne: Number(this.diffForm.amountOne) || 0,
                amountTwo: Number(this.diffForm.amountTwo) || 0,
                amountThree: Number(this.diffForm.amountThree) || 0,
            }).then(() => {
                this.$message.success('补差价成功')
                this.diffVisible = false
                this.diffLoading = false
                this.getList()
            }).catch(() => {
                this.diffLoading = false
            })
        },
        getList() {
            this.loading = true
            Gai.buy_list_4({
                page: this.current,
                ...this.searchData
            }).then((res) => {
                this.data = res.rewards.map((value, key) => {
                    return { ...value, key }
                })
                this.loading = false
                this.total = parseInt(res.count)
            })
        },
        subscription_delete(id, defaultValue) {
            let kkdt = 0;
            let amount = 0;

            this.$confirm({
                title: '删除确认',
                content: `确定要删除此条记录吗?`,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.sub_money({ id }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                    console.log('确认删除', id);
                }
                // content: () => (
                //     <div style={{ marginTop: '25px' }}>
                //         <a-row style={{ marginBottom: '15px' }}>
                //             <a-col span={6} style={{ height: '32px', lineHeight: '32px' }}>
                //                 返还金额
                //             </a-col>
                //             <a-col span={12}>
                //                 <a-input
                //                     placeholder="0"
                //                     onInput={(event) => {
                //                         amount = event.target.value; // 更新局部变量 amount
                //                     }}
                //                 />
                //             </a-col>
                //         </a-row>
                //         <a-row>
                //             <a-col span={6} style={{ height: '32px', lineHeight: '32px' }}>
                //                 扣除kkdt
                //             </a-col>
                //             <a-col span={12}>
                //                 <a-input
                //                     placeholder="0"
                //                     onInput={(event) => {
                //                         kkdt = event.target.value; // 更新局部变量 kkdt
                //                     }}
                //                 />
                //             </a-col>
                //         </a-row>
                //     </div>
                // ),
                // centered: true,
                // onOk: () => {
                //     return new Promise((resolve, reject) => {
                //         Gai.vip_delete({
                //             id,
                //             amount,
                //             kkdt
                //         }).then(res => {
                //             resolve()
                //             this.getList()
                //         }).catch(res => {
                //             reject()
                //         })
                //     });
                // }
            });
        }
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