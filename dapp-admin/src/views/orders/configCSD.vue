<template>
    <PageView>
        <a-card title="列表">
            <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="false" bordered
                :scroll="{ x: true }">
            </a-table>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'
export default {
    name: 'configCSD',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: '名称',
                    dataIndex: 'name',
                },
                {
                    title: '值',
                    dataIndex: 'value',
                },
                {
                    title: '操作',
                    key: 'action',
                    fixed: 'right',
                    width: 110,
                    customRender: (v) => {
                        return <a-button type="primary" onClick={() => {
                            this.config_update(v.id);
                        }}>修改</a-button>
                    },
                },
            ],
        }
    },
    methods: {
        getList() {
            this.loading = true
            Gai.config().then((res) => {
                this.data = res.config.filter(v=>v.id === "5").map((value, key) => {
                    return { ...value, key }
                })
                this.loading = false
                // this.total = parseInt(res.count)
            })
        },
        config_update(id) {
            let value = ""
            this.$confirm({
                title: `修改`,
                content: (
                    <a-input style="margin-top:25px;" placeholder="请输入" onInput={(val) => {
                        value = val.target.value
                    }} />
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.config_update({ id, value }).then(res => {
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
    >div {
        margin-bottom: 20px;
    }
}
</style>