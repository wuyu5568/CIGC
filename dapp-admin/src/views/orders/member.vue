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
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'

export default {
    name: 'member',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: '创建时间',
                    dataIndex: 'createdAt',
                    width: 170,
                },
                {
                    title: '用户地址',
                    dataIndex: 'address',
                    width: 180,
                    ellipsis: true,
                },
                {
                    title: '是否激活',
                    dataIndex: 'activated',
                    customRender: (v) => (v === true || v === '1' || v === 1) ? '是' : '否',
                },
                {
                    title: '账号锁定',
                    dataIndex: 'lock',
                    customRender: (v) => (v === '1' || v === true) ? '是' : '否',
                },
                {
                    title: '充值余额',
                    dataIndex: 'rechargeBalance',
                    customRender: (v) => v || '0',
                },
                {
                    title: '认购总金额',
                    dataIndex: 'paidAmount',
                    customRender: (v) => v || '0',
                },
                {
                    title: '购买代币总数量',
                    dataIndex: 'coinsTotal',
                    customRender: (v) => v || '0',
                },
                {
                    title: '代币每日释放量',
                    dataIndex: 'dailyCoins',
                    customRender: (v) => v || '0',
                },
                {
                    title: '已释放USDT',
                    dataIndex: 'releasedStatic',
                    customRender: (v) => v || '0',
                },
                {
                    title: '待释放静态',
                    dataIndex: 'pendingStatic',
                    customRender: (v) => v || '0',
                },
                {
                    title: '可提USDT',
                    dataIndex: 'available',
                    customRender: (v) => v || '0',
                },
                {
                    title: '可提ispay',
                    dataIndex: 'ispay',
                    customRender: (v) => v || '0',
                },
                {
                    title: '冻结USDT',
                    dataIndex: 'lockBalance',
                    customRender: (v) => v || '0',
                },
                {
                    title: '冻结代币',
                    dataIndex: 'lockIspay',
                    customRender: (v) => v || '0',
                },
                {
                    title: '有效日封顶',
                    dataIndex: 'capEffective',
                    customRender: (v) => v || '0',
                },
                {
                    title: '总业绩',
                    dataIndex: 'areaTotal',
                    customRender: (v) => v || '0',
                },
                {
                    title: '大区业绩',
                    dataIndex: 'areaMax',
                    customRender: (v) => v || '0',
                },
                {
                    title: '小区业绩',
                    dataIndex: 'areaMin',
                    customRender: (v) => v || '0',
                },
                {
                    title: '已对碰业绩',
                    dataIndex: 'pairedVolume',
                    customRender: (v) => v || '0',
                },
                {
                    title: '推荐人数',
                    dataIndex: 'historyRecommend',
                    customRender: (v) => v || '0',
                },
                {
                    title: '累计直推奖',
                    dataIndex: 'directTotal',
                    customRender: (v) => v || '0',
                },
                {
                    title: '累计对碰奖',
                    dataIndex: 'matchTotal',
                    customRender: (v) => v || '0',
                },
                {
                    title: '累计管理奖',
                    dataIndex: 'manageTotal',
                    customRender: (v) => v || '0',
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
                                    <a-button
                                        type="primary"
                                        onClick={() => {
                                            this.$router.push({ name: 'lookChildren', query: { userId: v.userId, address: v.address } })
                                        }}
                                    >
                                        查看下级
                                    </a-button>

                                    <a-dropdown>
                                        <a-button type="primary">
                                            更多
                                            <DownOutlined />
                                        </a-button>

                                        <a-menu slot="overlay">
                                            <a-menu-item onClick={() =>
                                                this.undo_lock(v.userId, v.lock === '1' ? '0' : '1')
                                            }>
                                                {v.lock === '1' ? '解锁' : '锁定'}
                                            </a-menu-item>

                                            <a-menu-item onClick={() => this.principal_update(v.address)}>
                                                设置可提余额
                                            </a-menu-item>

                                            <a-menu-item onClick={() => this.node_update(v.address)}>
                                                设置充值USDT
                                            </a-menu-item>

                                            <a-menu-item onClick={() => this.set_isPay(v.address)}>
                                                设置ispay
                                            </a-menu-item>
                                        </a-menu>
                                    </a-dropdown>
                                </a-button-group>
                            </div>
                        )
                    },
                },
            ],
            searchData: {
                address: '',
            },
        }
    },
    methods: {
        getList() {
            this.loading = true
            Gai.user_list({
                page: this.current,
                ...this.searchData
            }).then((res) => {
                this.data = res.users.map((value, key) => {
                    return { ...value, key }
                })
                this.loading = false
                this.total = parseInt(res.count)
            })
        },
        node_update(address) {
            let amount = ""
            this.$confirm({
                title: `设置充值USDT`,
                content: (
                    <a-input style="margin-top:25px;" placeholder="输入数量，正数增加、负数减少" onInput={(val) => {
                        amount = val.target.value
                    }} />
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.node_update({ address, usdt: amount }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        set_isPay(address) {
            let amount = ""
            this.$confirm({
                title: `设置ispay`,
                content: (
                    <a-input style="margin-top:25px;" placeholder="输入数量，正数增加、负数减少" onInput={(val) => {
                        amount = val.target.value
                    }} />
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.set_isPay({ address, amount: amount }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        principal_update(address) {
            let amount = ""
            this.$confirm({
                title: `设置可提余额`,
                content: (
                    <a-input style="margin-top:25px;" placeholder="输入数量，正数增加、负数减少" onInput={(val) => {
                        amount = val.target.value
                    }} />
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.principal_update({ address, usdt: amount }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        undo_lock(user_id, lock) {
            let text = lock === "1" ? `锁定` : `解锁`;
            this.$confirm({
                title: `${text}提示`,
                content: `确定要${text}此账户吗?`,
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.undo_lock({ user_id, lock }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
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
