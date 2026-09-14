<template>
    <PageView>
        <a-alert
            type="info"
            show-icon
            message="这里只改运营参数。热钱包私钥、创世地址、收款地址、打款开关、JWT、数据库连接不出现在本页。"
            style="margin-bottom:16px"
        />
        <a-spin :spinning="loading">
            <a-card v-for="g in groups" :key="g.title" :title="g.title" style="margin-bottom:16px">
                <a-table
                    :columns="columns"
                    :dataSource="g.rows"
                    :pagination="false"
                    bordered
                    :scroll="{ x: true }"
                    :rowKey="row => row.id"
                />
            </a-card>
        </a-spin>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'

const GROUP_ORDER = ['奖励', '提现', '价格', '冻结', '其他']

const META = {
    direct_rate: { group: '奖励', hint: '0.10 表示 10%。下级已支付订单金额 × 该比例。', effect: '之后新买单立刻生效；已发放的不追溯。' },
    match_rate: { group: '奖励', hint: '0.10 表示对碰额 pair 的 10%。', effect: '之后新对碰立刻生效；已发放的不追溯。' },
    manage_rate: { group: '奖励', hint: '对碰产值 × 该比例为管理奖总池，再按代数均分。', effect: '之后新对碰立刻生效；已发放的不追溯。' },
    manage_generations: { group: '奖励', hint: '沿邀请链向上找几代已激活用户（1–10）。未激活跳过继续往上。', effect: '之后新发的管理奖按新代数；已发的份额不追回。' },
    min_withdraw_amount: { group: '提现', hint: 'USDT 单笔最低申请额，单位 U。', effect: '之后新申请生效。' },
    min_withdraw_amount_ispay: { group: '提现', hint: 'ISPAY 单笔最低申请额；0 表示只需大于 0。', effect: '之后新申请生效。' },
    withdraw_fee_rate: { group: '提现', hint: '0.10 表示 10%。USDT 到账 = 申请额 − 手续费。', effect: '之后新申请生效。' },
    withdraw_fee_rate_ispay: { group: '提现', hint: '0 表示免手续费。ISPAY 到账 = 申请额 − 手续费。', effect: '之后新申请生效。链上打款仍未开通。' },
    withdraw_daily_limit: { group: '提现', hint: '上海自然日 USDT 累计申请上限；0 不限制。', effect: '立刻按当天已申请额计算剩余。' },
    withdraw_daily_limit_ispay: { group: '提现', hint: '上海自然日 ISPAY 累计申请上限；0 不限制。', effect: '立刻按当天已申请额计算剩余。' },
    ispay_price: { group: '价格', hint: '测试用交易所现价（U）。拆一半 U / 一半 ispay 时用。', effect: '立刻影响新入账、待释放展示。不是链上真实行情。' },
    overflow_clear_hours: { group: '冻结', hint: '封账（次日 0:00）后 N 小时清除超额/未激活冻结。默认 72。只减冻结、不转可提。', effect: '只影响之后新封账的批次；已经盖了到期日的批次不变。' },
}

function withMeta(row) {
    const m = META[row.key] || {}
    return {
        ...row,
        group: row.group || m.group || '其他',
        hint: row.hint || m.hint || '',
        effect: row.effect || m.effect || '保存后生效。',
    }
}

export default {
    name: 'config',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: '名称',
                    dataIndex: 'name',
                    width: 180,
                },
                {
                    title: '键',
                    dataIndex: 'key',
                    width: 220,
                },
                {
                    title: '当前值',
                    dataIndex: 'value',
                    width: 120,
                },
                {
                    title: '说明',
                    dataIndex: 'hint',
                    customRender: (v) => <span style="white-space:normal">{v}</span>,
                },
                {
                    title: '何时生效',
                    dataIndex: 'effect',
                    width: 260,
                    customRender: (v) => <span style="white-space:normal">{v}</span>,
                },
                {
                    title: '操作',
                    key: 'action',
                    fixed: 'right',
                    width: 110,
                    customRender: (v) => {
                        return <a-button type="primary" onClick={() => this.config_update(v)}>修改</a-button>
                    },
                },
            ],
        }
    },
    computed: {
        groups() {
            const buckets = {}
            GROUP_ORDER.forEach((title) => { buckets[title] = [] })
            this.data.forEach((row) => {
                const title = row.group || '其他'
                if (!buckets[title]) buckets[title] = []
                buckets[title].push(row)
            })
            const extra = Object.keys(buckets).filter((k) => GROUP_ORDER.indexOf(k) < 0 && buckets[k].length)
            return GROUP_ORDER.concat(extra)
                .filter((title) => buckets[title] && buckets[title].length)
                .map((title) => ({ title, rows: buckets[title] }))
        },
    },
    methods: {
        getList() {
            this.loading = true
            Gai.config().then((res) => {
                this.data = (res.config || []).map((value, key) => {
                    return withMeta({ ...value, key })
                })
                this.loading = false
            }).catch(() => {
                this.loading = false
            })
        },
        config_update(row) {
            let value = row.value
            this.$confirm({
                title: `修改「${row.name}」`,
                content: (
                    <div>
                        <p style="margin:0 0 8px;color:#595959">{row.hint}</p>
                        <p style="margin:0 0 12px;color:#8c8c8c;font-size:12px">生效：{row.effect}</p>
                        <a-input defaultValue={row.value} placeholder="请输入" onInput={(e) => { value = e.target.value }} />
                    </div>
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.config_update({ id: row.id, value }).then(() => {
                            resolve()
                            this.getList()
                        }).catch(() => {
                            reject()
                        })
                    })
                },
            })
        },
    },
}
</script>
