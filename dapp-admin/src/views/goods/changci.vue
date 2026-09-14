<template>
    <PageView>
        <a-card title="列表">
            <a-table
                :loading="loading"
                :columns="columns"
                :dataSource="data"
                :pagination="{ total, pageSize, showSizeChanger, current }"
                @change="changePagination"
                bordered
                :scroll="{ x: true }"
            >
            </a-table>
        </a-card>
        <a-modal
            title="场次信息"
            :visible="isShow"
            centered
            :closable="false"
            :maskClosable="false"
            width="360px"
        >
            <a class="radioTitle">开始时间</a>
            <a-input v-model="form.start" placeholder="开始时间"/>
            <a class="radioTitle">结束时间</a>
            <a-input v-model="form.end" placeholder="结束时间"/>
            <a class="radioTitle">状态(1开启,0关闭)</a>
            <a-input v-model="form.status" placeholder="开始时间"/>
            <template slot="footer">
                <a-button-group>
                    <a-button type="primary" :loading="btnloading" @click="updateProjectType()">确定</a-button>
                    <a-button type="default" @click="isShow = false">关闭</a-button>
                </a-button-group>
            </template>
        </a-modal>
    </PageView>
</template>

<script type="text/jsx">
import Project from '../../api/Project'
import listMixin from '../mixin/listMixin'
export default {
    name: 'changci',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: '名称',
                    dataIndex: 'name',
                    customRender: (v) => <a>{v}</a>,
                },
                {
                    title: '开始时间',
                    dataIndex: 'start',
                },
                {
                    title: '结束时间',
                    dataIndex: 'end',
                },
                {
                    title: '状态',
                    dataIndex: 'status',
                    customRender: (v) => v==="0"?`已关闭`:`已开启`,
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
                                    <a-menu-item
                                        onClick={() => {
                                            this.form = {
                                                id:v.id,
                                                start:v.start,
                                                end:v.end,
                                                status:v.status,
                                            }
                                            this.isShow = true;
                                        }}
                                    >
                                        修改信息
                                    </a-menu-item>
                                </a-menu>
                                <a-button>
                                    操作 <a-icon type="down" />
                                </a-button>
                            </a-dropdown>
                        )
                    },
                },
            ],
            isShow:false,
            form:{
                id:undefined,
                start:undefined,
                end:undefined,
                status:undefined
            },
            btnloading:false,
        }
    },
    methods: {
        getList() {
            this.loading = true
            Project.getProjectType().then((res) => {
                this.data = res.data.map((value, key) => {
                    return { ...value, key }
                })
                this.loading = false
                this.total = parseInt(res.count)
            })
        },
        updateProjectType(){
            this.btnloading = true
                this.$nextTick(()=>{
                    Project.updateProjectType(this.form).then(res => {
                        this.isShow = false
                        this.btnloading = false
                        this.getList()
                    }).catch(res=>{
                        this.isShow = false
                        this.btnloading = false
                    })
                })
        }
    },
}
</script>

<style scoped lang="less">
.inputGroup {
    > div {
        margin-bottom: 20px;
    }
}
.radioTitle {
    display: block;
    font-size: 18px;
    padding: 20px 0px;
}
.radioStyle {
    display: block;
    height: 30px;
    line-height: 30px;
    margin-left: 20px;
}
</style>