<template>
    <PageView>
        <a-card title="列表">
            <a-button type="primary" slot="extra" @click="$router.push({name:`newsEdit`})">添加公告</a-button>
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
    import Art from '../../api/Art'
    import listMixin from '../mixin/listMixin'
    export default {
        name: 'news',
        mixins: [listMixin],
        data () {
            return {
                columns: [
                    {
                        title: '标题',
                        dataIndex: 'title',
                    },
                    {
                        title: '添加时间',
                        dataIndex: 'add_time',
                        customRender: (v) => this.timeOne(v)
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
                                            this.changeBanner(v.id)
                                        }}>删除
                                        </a-menu-item>
                                        <a-menu-item onClick={() => {
                                            this.$router.push({name:"newsEdit",query:{id:v.id}})
                                        }}>编辑
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
                Art.getArticle({
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
            changeBanner (id) {
                this.$confirm({
                    title: `删除提示`,
                    content: `确定要删除此公告吗?`,
                    centered: true,
                    onOk: () => {
                        return new Promise((resolve, reject) => {
                            Art.deleteArticle({ id }).then(res => {
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