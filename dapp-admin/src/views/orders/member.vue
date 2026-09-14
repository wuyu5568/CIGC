<template>
    <PageView>
        <a-card title="列表">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="账户" @keyup.enter="getListTwo" />
                </a-col>
                <!-- <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.vip" style="width:100%" placeholder="级别" @change="getListTwo">
                        <a-select-option value="1000">1000</a-select-option>
                        <a-select-option value="3000">3000</a-select-option>
                        <a-select-option value="5000">5000</a-select-option>
                        <a-select-option value="10000">10000</a-select-option>
                        <a-select-option value="15000">15000</a-select-option>
                        <a-select-option value="30000">30000</a-select-option>
                    </a-select>
                </a-col> -->
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
                },
                {
                    title: '地址',
                    dataIndex: 'address',
                },
                {
                    title: '当前认购金额',
                    dataIndex: 'amountUsdtCurrent',
                },
                {
                    title: 'brc20余额',
                    dataIndex: 'bAmount'
                },
                {
                    title: '已释放',
                    dataIndex: 'amountUsdtGet',
                },
                {
                    title: '待释放',
                    dataIndex: 'amountUsdtTwo',
                },
                {
                    title: 'usdt数量',
                    dataIndex: 'balanceUsdt',
                },
                {
                    title: '收益冻结',
                    dataIndex: 'lockBalance',
                    customRender: (v) => v || '0',
                },
                {
                    title: '收益冻结 ispay',
                    dataIndex: 'lockIspay',
                    customRender: (v) => v || '0',
                },
                {
                    title: 'ispay预售1',
                    dataIndex: 'balanceDhb',
                },
                {
                    title: 'ispay预售2',
                    dataIndex: 'bAmountTwo',
                },
                {
                    title: '预售2每日释放数量',
                    dataIndex: 'perDayAmount',
                },
                {
                    title: '出局次数',
                    dataIndex: 'out',
                },
                {
                    title: '总业绩',
                    dataIndex: 'areaTotal',
                },
                {
                    title: '小区',
                    dataIndex: 'areaMin',
                },
                {
                    title: '大区',
                    dataIndex: 'areaMax',
                },
                {
                    title: '级别',
                    dataIndex: 'vip',
                },
                {
                    title: '历史推荐人数',
                    dataIndex: 'historyRecommend',
                },
                {
                    title: '是否锁定',
                    dataIndex: 'lock',
                    customRender: (v) => {
                        if (v === '1') return '是'
                        return '否'
                    }
                },
                {
                    title: '上级分红',
                    dataIndex: 'lockReward',
                    customRender: (v) => {
                        if (v === '0') return '开启'
                        return '关闭'
                    }
                },
                {
                    title: '上级地址',
                    dataIndex: 'myRecommendAddress',
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
                                            <a-menu-item onClick={() => this.vip_update(v.userId, v.vip)}>
                                                修改级别
                                            </a-menu-item>

                                            <a-menu-item onClick={() =>
                                                this.undo_lock(v.userId, v.lock === '1' ? '0' : '1')
                                            }>
                                                {v.lock === '1' ? '解锁' : '锁定'}
                                            </a-menu-item>

                                            <a-menu-item onClick={() =>
                                                this.undo_all_lock(v.userId, v.lock === '1' ? '0' : '1')
                                            }>
                                                {v.lock === '1' ? '解锁一条线' : '锁定一条线'}
                                            </a-menu-item>

                                            <a-menu-item onClick={() =>
                                                this.dividend_policy(v.userId, v.lockReward === '1' ? '0' : '1')
                                            }>
                                                {v.lockReward === '1' ? '开启上级分红' : '关闭上级分红'}
                                            </a-menu-item>

                                            <a-menu-divider />

                                            <a-menu-item onClick={() => this.set_isPay(v.address)}>
                                                设置brc20余额
                                            </a-menu-item>

                                            <a-menu-item onClick={() => this.principal_update(v.address)}>
                                                设置可提余额
                                            </a-menu-item>

                                            <a-menu-item onClick={() => this.node_update(v.address)}>
                                                设置充值USDT
                                            </a-menu-item>

                                            <a-menu-item onClick={() => this.setOrder(v.address)}>
                                                设置预售2订单
                                            </a-menu-item>

                                            <a-menu-item onClick={() => this.edit_address(v.address)}>
                                                修改地址
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
                vip: undefined,
                isLocation: undefined
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
        set_password(user_id) {
            let pass = ""
            this.$confirm({
                title: `设置密码`,
                content: (
                    <a-input style="margin-top:25px;" type="password" placeholder="请输入密码" onInput={(val) => {
                        pass = val.target.value
                    }} />
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.set_pass({ user_id, pass }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        vip_update(user_id, defaultValue) {
            // if (defaultValue === '0') {
            //     defaultValue = undefined
            // }
            let vip = defaultValue;
            this.$confirm({
                title: `修改VIP级别`,
                content: (
                    <a-select style="width:200px" defaultValue={defaultValue} placeholder="选择级别" onChange={(val) => {
                        vip = val;
                    }}>
                        <a-select-option value="0">无VIP级别</a-select-option>
                        <a-select-option value="1">1</a-select-option>
                        <a-select-option value="2">2</a-select-option>
                        <a-select-option value="3">3</a-select-option>
                        <a-select-option value="4">4</a-select-option>
                        <a-select-option value="5">5</a-select-option>
                    </a-select>
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        if (vip === undefined) {
                            this.$notification.warning({
                                message: '提示',
                                description: '请选择级别'
                            })
                            reject()
                            return;
                        }
                        Gai.vip_update({ user_id, vip }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
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
                title: `设置brc20余额`,
                content: (
                    <a-input style="margin-top:25px;" placeholder="输入数量" onInput={(val) => {
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
                    <a-input style="margin-top:25px;" placeholder="输入数量" onInput={(val) => {
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
        setOrder(address) {
            let amount = ""
            this.$confirm({
                title: `设置金额`,
                content: (
                    <a-input style="margin-top:25px;" placeholder="输入金额" onInput={(val) => {
                        amount = val.target.value
                    }} />
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.order_buy_four({ address, amount }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        balance_update(user_id, defaultValue) {
            let amount = ""
            this.$confirm({
                title: `设置基金`,
                content: (
                    <a-input style="margin-top:25px;" placeholder="请输入" onInput={(val) => {
                        amount = val.target.value
                    }} />
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.balance_update({ user_id, amount }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        admin_update_location_new_max(user_id) {
            let amount = ""
            this.$confirm({
                title: `修改待释放余额`,
                content: (
                    <a-input style="margin-top:25px;" placeholder="请输入" onInput={(val) => {
                        amount = val.target.value
                    }} />
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.admin_update_location_new_max({ user_id, amount }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        password_update(user_id) {
            let password = ""
            this.$confirm({
                title: `修改密码`,
                content: (
                    <a-input style="margin-top:25px;" placeholder="请输入秘钥" onInput={(val) => {
                        password = val.target.value
                    }} />
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.password_update({ user_id, password }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        location_insert(user_id) {
            let amount = undefined;
            this.$confirm({
                title: `入单充值`,
                content: (
                    <a-select style="width:200px" placeholder="选择金额" onChange={(val) => {
                        amount = val;
                    }}>
                        <a-select-option value="50">50</a-select-option>
                        <a-select-option value="100">100</a-select-option>
                        <a-select-option value="300">300</a-select-option>
                    </a-select>
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        if (amount === undefined) {
                            this.$notification.warning({
                                message: '提示',
                                description: '请选择金额'
                            })
                            reject()
                            return;
                        }
                        Gai.location_insert({ user_id, amount }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        undo_update(user_id, undo) {
            let text = undo === "1" ? `冻结` : `解冻`;
            this.$confirm({
                title: `${text}提示`,
                content: `确定要${text}此账户吗?`,
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.undo_update({ user_id, undo }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        recommend_level(user_id, level) {
            let text = level === "1" ? `开启特殊奖励` : `关闭特殊奖励`;
            this.$confirm({
                title: `${text}提示`,
                content: `确定要${text}特殊奖励?`,
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.recommend_level({ user_id, level }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        dividend_policy(user_id, lockReward) {
            let text = lockReward === "1" ? `开启上级分红` : `关闭上级分红`;
            this.$confirm({
                title: `${text}提示`,
                content: `确定要${text}?`,
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.dividend_policy({ user_id, lockReward: Number(lockReward) }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        undo_all_lock(user_id, lock) {
            let text = lock === "1" ? `锁定一条线` : `解锁一条线`;
            this.$confirm({
                title: `${text}提示`,
                content: `确定要${text}吗?`,
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.undo_lock({ user_id, one: '1', lock }).then(res => {
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
        edit_address(id) {
            let value = ""
            this.$confirm({
                title: `修改地址`,
                content: (
                    <a-input style="margin-top:25px;" placeholder="请输入地址" onInput={(val) => {
                        value = val.target.value
                    }} />
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.change_address({ address: id, addressTwo: value }).then(res => {
                            resolve()
                            this.getList()
                        }).catch(res => {
                            reject()
                        })
                    })
                }
            })
        },
        level_update(user_id) {
            let level = undefined;
            this.$confirm({
                title: `修改节点等级`,
                content: (
                    <a-select style="width:200px" placeholder="选择节点等级" onChange={(val) => {
                        level = val;
                    }}>
                        <a-select-option value="1">节点</a-select-option>
                        <a-select-option value="2">普通节点</a-select-option>
                        <a-select-option value="3">超级节点</a-select-option>
                        <a-select-option value="4">创世节点</a-select-option>
                    </a-select>
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        if (level === undefined) {
                            this.$notification.warning({
                                message: '提示',
                                description: '请选择等级'
                            })
                            reject()
                            return;
                        }
                        Gai.level_update({ user_id, level }).then(res => {
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