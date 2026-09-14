<template>
    <PageView>
        <a-card title="提现审核">
            <div class="payout-bar">
                <span>USDT 申请后直接进打款队列，每分钟自动打款。ISPAY 仍需审核。</span>
                <span>打款：{{ payoutEnabled ? '已开启' : '未开启' }}</span>
                <span v-if="hotWallet">热钱包：{{ hotWallet }}</span>
                <span v-if="payoutMaxUsdt">单笔上限：{{ payoutMaxUsdt }} USDT</span>
                <a-button type="primary" :disabled="!payoutEnabled" :loading="payoutLoading" @click="payoutAll">处理打款队列</a-button>
            </div>
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="账户" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.withDrawType" style="width:100%" placeholder="类型"
                        @change="getListTwo">
                        <a-select-option value="USDT">USDT</a-select-option>
                        <a-select-option value="RAW_NEW">ISPAY</a-select-option>
                    </a-select>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select v-model="searchData.status" style="width:100%" placeholder="状态" @change="getListTwo">
                        <a-select-option value="pending">待审核</a-select-option>
                        <a-select-option value="rewarded">已通过待打款</a-select-option>
                        <a-select-option value="doing">打款中</a-select-option>
                        <a-select-option value="pass">已打款</a-select-option>
                        <a-select-option value="rejected">已拒绝</a-select-option>
                        <a-select-option value="cancelled">已取消</a-select-option>
                        <a-select-option value="">全部</a-select-option>
                    </a-select>
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

const statusText = {
    pending: '待审核',
    rewarded: '已通过待打款',
    doing: '打款中',
    pass: '已打款',
    rejected: '已拒绝',
    cancelled: '已取消',
}

export default {
    name: 'withdraw',
    mixins: [listMixin],
    data() {
        return {
            payoutEnabled: false,
            hotWallet: '',
            payoutMaxUsdt: '',
            payoutLoading: false,
            columns: [
                {
                    title: '账户',
                    dataIndex: 'address',
                },
                {
                    title: '类型',
                    dataIndex: 'asset',
                    customRender: (v) => (v === 'ispay' ? 'ISPAY' : 'USDT'),
                },
                {
                    title: '提现金额',
                    dataIndex: 'amount',
                },
                {
                    title: '手续费',
                    dataIndex: 'feeAmount',
                },
                {
                    title: '到账金额',
                    dataIndex: 'relAmount',
                },
                {
                    title: '状态',
                    dataIndex: 'status',
                    customRender: (v) => statusText[v] || v,
                },
                {
                    title: '交易哈希',
                    dataIndex: 'txHash',
                    customRender: (v) => {
                        if (!v) return ''
                        return <a href={`https://bscscan.com/tx/${v}`} target="_blank">{v.slice(0, 10)}...</a>
                    },
                },
                {
                    title: '打款说明',
                    dataIndex: 'payoutError',
                },
                {
                    title: '创建时间',
                    dataIndex: 'createdAt',
                },
                {
                    title: '操作',
                    key: 'action',
                    fixed: 'right',
                    width: 180,
                    customRender: (v) => {
                        const btns = []
                        if (v.status === 'pending') {
                            btns.push(<a-button type="primary" size="small" onClick={() => this.pass(v.id)}>通过</a-button>)
                            btns.push(<a-button size="small" onClick={() => this.reject(v.id)}>拒绝</a-button>)
                        }
                        if ((v.status === 'rewarded' || v.status === 'doing') && v.asset !== 'ispay') {
                            btns.push(<a-button type="primary" size="small" disabled={!this.payoutEnabled} onClick={() => this.payoutOne(v.id)}>打款</a-button>)
                        }
                        if (!btns.length) return ''
                        return <a-button-group>{btns}</a-button-group>
                    },
                },
            ],
            searchData: {
                address: '',
                withDrawType: undefined,
                status: 'rewarded',
            },
        }
    },
    methods: {
        getList() {
            this.loading = true
            Gai.withdraw_list({
                page: this.current,
                ...this.searchData
            }).then((res) => {
                this.data = (res.withdraw || []).map((value, key) => {
                    return { ...value, key }
                })
                this.payoutEnabled = !!res.payoutEnabled
                this.hotWallet = res.hotWallet || ''
                this.payoutMaxUsdt = res.payoutMaxUsdt || ''
                this.loading = false
                this.total = parseInt(res.count)
            }).catch(() => {
                this.loading = false
            })
        },
        pass(id) {
            this.$confirm({
                title: '通过提示',
                content: '确定要通过此次提现吗？通过后进入打款队列（USDT 申请时已自动入队）。',
                centered: true,
                onOk: () => {
                    return Gai.withdraw_pass({ id }).then(() => {
                        this.$message.success('已通过')
                        this.getList()
                    })
                }
            })
        },
        reject(id) {
            this.$confirm({
                title: '拒绝提示',
                content: '确定要拒绝此次提现吗？金额将退回用户可提现余额。',
                centered: true,
                onOk: () => {
                    return Gai.withdraw_reject({ id }).then(() => {
                        this.$message.success('已拒绝')
                        this.getList()
                    })
                }
            })
        },
        payoutOne(id) {
            this.$confirm({
                title: '打款提示',
                content: '将用热钱包向该用户地址打 USDT，确定继续？',
                centered: true,
                onOk: () => this.runPayout({ id })
            })
        },
        payoutAll() {
            this.$confirm({
                title: '打款队列',
                content: '处理当前已通过/打款中的 USDT 提现，确定继续？',
                centered: true,
                onOk: () => this.runPayout({})
            })
        },
        runPayout(payload) {
            this.payoutLoading = true
            return Gai.withdraw_payout(payload).then((res) => {
                this.payoutLoading = false
                if (res.status !== 'ok') {
                    this.$message.error(res.status || '打款失败')
                    return
                }
                this.$message.success(`扫描${res.scanned || 0} 发出${res.sent || 0} 完成${res.passed || 0} 失败${res.failed || 0}`)
                this.getList()
            }).catch((err) => {
                this.payoutLoading = false
                this.$message.error((err && err.message) || '打款失败')
            })
        },
    },
}
</script>

<style scoped lang="less">
.payout-bar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
    word-break: break-all;
}
.inputGroup {
    >div {
        margin-bottom: 20px;
    }
}
</style>
