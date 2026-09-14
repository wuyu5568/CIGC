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
    import Log from '../../api/Log'
    import listMixin from '../mixin/listMixin'
    export default {
        name: 'cz',
        mixins:[listMixin],
        data(){
            return{
                columns: [
                    {
                        title: 'IP',
                        dataIndex: 'ip',
                        customRender: (v) => <a>{v}</a>
                    },
                    {
                        title: '操作内容',
                        dataIndex: 'content',
                    },
                    {
                        title: '时间',
                        dataIndex: 'dotime',
                        customRender: (v) => this.timeOne(v)
                    },
                ],
            }
        },
        methods: {
            getList () {
                this.loading = true
                Log.getCZlog({
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