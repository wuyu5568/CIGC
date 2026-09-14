<template>
    <PageView>
        <a-card title="用户组列表">
            <a-button type="primary" slot="extra" @click="$router.push({name:`addUser`})">新增用户组</a-button>
            <a-table
                :loading="loading"
                :columns="columns"
                :dataSource="data"
                :pagination="false"
                bordered :scroll="{x:true}">
            </a-table>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
    import Auth from '../../api/Auth'

    export default {
        name: 'user',
        data () {
            return {
                loading: true,
                columns: [
                    {
                        title: 'ID',
                        dataIndex: 'id',
                        customRender: (v) => <a>{v}</a>
                    },
                    {
                        title: '名称',
                        dataIndex: 'title',
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
                                            this.lookPower(v.id)
                                        }}>查看权限
                                        </a-menu-item>
                                        <a-menu-item onClick={() => {
                                            this.changeName(v.id)
                                        }}>修改名称
                                        </a-menu-item>
                                        <a-menu-item onClick={() => {
                                            this.delGroup(v.id)
                                        }}>删除
                                        </a-menu-item>
                                    </a-menu>
                                    <a-button>操作 <a-icon type="down"/></a-button>
                                </a-dropdown>
                            )
                        }
                    }
                ],
                data: []
            }
        },
        activated () {
            this.getList()
        },
        methods: {
            getList () {
                this.loading = true
                Auth.getGroup().then(res => {
                    this.data = res.group.map((value, key) => {
                        return { ...value, key }
                    })
                    this.loading = false
                })
            },
            delGroup(id) {
                this.$confirm({
                    title: `删除`,
                    content: `确定要删除此用户组吗?`,
                    centered: true,
                    onOk: () => {
                        return new Promise((resolve, reject) => {
                            Auth.deleteGroup({id}).then(res => {
                                resolve()
                                this.getList()
                            }).catch(res => {
                                reject()
                            })
                        })
                    }
                })
            },
            changeName(id){
                let title = ""
                this.$confirm({
                    title: `修改名称`,
                    content: (
                        <a-input style="margin-top:25px;" placeholder="请输入名称" onInput={(val) => {
                            title = val.target.value
                        }}/>
                    ),
                    centered: true,
                    onOk: () => {
                        return new Promise((resolve, reject) => {
                            if (title === '') {
                                this.$notification.warning({
                                    message: '提示',
                                    description: '请输入名称'
                                })
                                reject()
                            } else {
                                Auth.changeGroup({ id,title }).then(res => {
                                    resolve()
                                    this.getList()
                                }).catch(res => {
                                    reject()
                                })
                            }
                        })
                    }
                })
            },
            lookPower(id){
                Auth.getGroupDetail({id}).then(res => {
                    let checkIdArr = []
                    let inputArray = res.data.map((v1,k1)=>{
                        v1.key = v1.id
                        if(v1.is_have)checkIdArr.push(v1.id)
                        v1.children = v1.son.map((v2,k2)=>{
                            v2.key = v2.id
                            if(v2.is_have)checkIdArr.push(v2.id)
                            v2.children = []
                            return v2
                        })
                        return v1
                    })
                    this.$confirm({
                        title: `查看权限`,
                        content: (
                            <a-tree checkable treeData={inputArray} defaultCheckedKeys={checkIdArr} onCheck={(checkedKeys)=>{checkIdArr = checkedKeys}}></a-tree>
                        ),
                        centered: true,
                        okText:"修改",
                        onOk: () => {
                            return new Promise((resolve, reject) => {
                                Auth.changeGroup({id,rule:checkIdArr.join(",")}).then(res => {
                                    resolve()
                                    this.getList()
                                }).catch(res => {
                                    reject()
                                })
                            })
                        }
                    })
                })
            }
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