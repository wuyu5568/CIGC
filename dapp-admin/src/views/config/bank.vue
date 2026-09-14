<template>
    <PageView>
        <a-card title="列表">
            <a-table
                :loading="loading"
                :columns="columns"
                :dataSource="data"
                :pagination="{total,pageSize,showSizeChanger,current}"
                @change="changePagination"
                bordered :scroll="{x:true}">
            </a-table>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
    import Bank from '../../api/Bank'
    import listMixin from '../mixin/listMixin'
    export default {
        name: 'bank',
        mixins: [listMixin],
        data () {
            return {
                columns: [
                    {
                        title: '名称',
                        dataIndex: 'bankname',
                    },
                    {
                        title: '状态',
                        dataIndex: 'status',
                        customRender: (v) => {
                            let status = {
                                1: { name: '显示', color: 'success' },
                                0: { name: '隐藏', color: 'error' }
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
                                        <a-menu-item onClick={() => {
                                            this.updateBankStatus(v)
                                        }}>{v.status === `1`?`隐藏`:`显示`}
                                        </a-menu-item>
                                    </a-menu>
                                    <a-button>操作 <a-icon type="down"/></a-button>
                                </a-dropdown>
                            )
                        }
                    }
                ],
            }
        },
        methods: {
            getList () {
                this.loading = true
                Bank.getBankList({
                    page: this.current,
                    num: this.pageSize,
                }).then(res => {
                    this.data = res.data.map((value, key) => {
                        return { ...value, key }
                    })
                    this.loading = false
                    this.total = parseInt(res.count)
                })
            },
            updateBankStatus (v) {
                this.$confirm({
                    title: `${v.status === `1`?`隐藏`:`显示`}提示`,
                    content: `确定要${v.status === `1`?`隐藏`:`显示`}此支付方式吗?`,
                    centered: true,
                    onOk: () => {
                        return new Promise((resolve, reject) => {
                            Bank.updateBankStatus({ id:v.id }).then(res => {
                                resolve()
                                this.getList()
                            }).catch(res => {
                                reject()
                            })
                        })
                    }
                })
            },
        }
    }
</script>

<style scoped lang="less">
    .inputGroup {
        > div {
            margin-bottom: 20px;
        }
    }
</style>