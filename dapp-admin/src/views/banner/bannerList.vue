<template>
    <PageView>
        <a-card title="列表">
            <a-button type="primary" slot="extra" @click="$router.push({name:`bannerEdit`})">添加轮播</a-button>
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
        name: 'bannerList',
        mixins: [listMixin],
        data () {
            return {
                columns: [
                    {
                        title: '图片',
                        dataIndex: 'image',
                        customRender:(v)=>{
                            return <img style="display:block;height:40px" src={projectUrl+v}/>
                        }
                    },
                    {
                        title: '标题',
                        dataIndex: 'title',
                    },
                    {
                        title: '状态',
                        dataIndex: 'status',
                        customRender: (v) => {
                            let status = {
                                1: { name: '显示', color: 'success' },
                                2: { name: '隐藏', color: 'error' }
                            }
                            return <a-badge status={status[v].color} text={status[v].name}/>
                        }
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
                                            this.$router.push({name:"bannerEdit",query:{id:v.id}})
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
                Art.getAllArticle({
                    page: this.current,
                    num: this.pageSize,
                    type:1,
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
                    content: `确定要删除此轮播吗?`,
                    centered: true,
                    onOk: () => {
                        return new Promise((resolve, reject) => {
                            Art.delArticle({ id }).then(res => {
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