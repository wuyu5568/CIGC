<template>
    <PageView>
        <a-card title="列表">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.username" placeholder="请输入钱包地址" @keyup.enter="getListTwo"/>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.level" style="width:100%" placeholder="等级筛选"
                              @change="getListTwo">
                        <a-select-option value="0">0星</a-select-option>
                        <a-select-option value="1">1星</a-select-option>
                        <a-select-option value="2">2星</a-select-option>
                        <a-select-option value="3">3星</a-select-option>
                        <a-select-option value="4">4星</a-select-option>
                        <a-select-option value="5">5星</a-select-option>
                    </a-select>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-button-group>
                        <a-button type="primary" :loading="loading" @click="getListTwo">确定筛选</a-button>
                    </a-button-group>
                </a-col>
            </a-row>
            <a-table
                :loading="loading"
                :columns="columns"
                :dataSource="data"
                :pagination="{total,pageSize,showSizeChanger,current}"
                @change="changePagination"
                bordered :scroll="{x:true}">
            </a-table>
        </a-card>
        <!--修改金额-->
        <a-modal
            title="金额"
            :visible="isShow"
            centered
            :closable="false"
            :maskClosable="false"
            width="360px"
        >
            <a class="radioTitle">类型</a>
            <a-radio-group v-model="moneyType">
                <a-radio class="radioStyle" :value="1">USDT</a-radio>
                <a-radio class="radioStyle" :value="2">TDH</a-radio>
                <a-radio class="radioStyle" :value="3">DFCC</a-radio>
            </a-radio-group>
            <a class="radioTitle">修改金额</a>
            <a-input-number :min="0" :max="100000000" style="margin-left: 20px;width:120px;" v-model="num"/>
            <template slot="footer">
                <a-button-group>
                    <a-button type="primary" :loading="btnloading" @click="addCoin(1)">增加</a-button>
                    <a-button type="primary" :loading="btnloading" @click="addCoin(2)">减少</a-button>
                    <a-button type="default" @click="isShow = false">关闭</a-button>
                </a-button-group>
            </template>
        </a-modal>
    </PageView>
</template>

<script type="text/jsx">
    import Member from '../../api/Member'
    import listMixin from '../mixin/listMixin'
    export default {
        name: 'memberList',
        mixins:[listMixin],
        data(){
            return{
                columns: [
                    {
                        title: '地址',
                        dataIndex: 'username',
                        customRender: (v) => <a>{v}</a>
                    },
                    {
                        title: '级别',
                        dataIndex: 'level',
                    },
                    {
                        title: 'USDT',
                        dataIndex: 'money1',
                        customRender: (v) => v||0
                    },
                    {
                        title: 'TDH',
                        dataIndex: 'money2',
                        customRender: (v) => v||0
                    },
                    {
                        title: 'DFCC',
                        dataIndex: 'money3',
                        customRender: (v) => v||0
                    },
                    {
                        title: 'TDHT',
                        dataIndex: 'money4',
                        customRender: (v) => v||0
                    },
                    {
                        title: '注册时间',
                        dataIndex: 'add_time',
                        customRender: (v) => v?this.timeOne(v):"无",
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
                                            this.uid = v.uid
                                            this.num = 0
                                            this.moneyType = 1
                                            this.isShow = true
                                        }}>修改金额
                                        </a-menu-item>
                                        <a-menu-item onClick={() => {
                                            this.$router.push({name:"addMember",query:{uid:v.uid}})
                                        }}>修改信息
                                        </a-menu-item>
                                        <a-menu-item onClick={() => {
                                            this.$router.push({name:"addGmOrder",query:{uid:v.uid}})
                                        }}>添加挂卖单
                                        </a-menu-item>
                                        <a-menu-item onClick={() => {
                                            this.addNodeLog(v.uid)
                                        }}>添加节点账户
                                        </a-menu-item>
                                    </a-menu>
                                    <a-button>操作 <a-icon type="down"/></a-button>
                                </a-dropdown>
                            )
                        }
                    }
                ],
                searchData: {
                    username: "",
                    level: undefined,
                },
                isShow:false,
                moneyType:1,
                num:0,
                btnloading:false,
                uid:undefined,
            }
        },
        methods: {
            getList () {
                this.loading = true
                Member.index({
                    page: this.current,
                    num: this.pageSize,
                    ...this.searchData
                }).then(res => {
                    this.data = res.data.map((value, key) => {
                        return { ...value, key }
                    })
                    this.loading = false
                    this.total = parseInt(res.count)
                })
            },
            addCoin(sz_type){
                this.btnloading = true
                this.$nextTick(()=>{
                    Member.runUpdateUserMoney({ uid:this.uid,type:this.moneyType,sz_type,num:this.num}).then(res => {
                        this.isShow = false
                        this.btnloading = false
                        this.getList()
                    }).catch(res=>{
                        this.isShow = false
                        this.btnloading = false
                    })
                })
            },
            addNodeLog(uid){
                let money = ""
                this.$confirm({
                    title: `代币数量`,
                    content: (
                        <a-input style="margin-top:25px;" placeholder="请输入代币数量" onInput={(val) => {
                            money = val.target.value
                        }}/>
                    ),
                    centered: true,
                    onOk: () => {
                        return new Promise((resolve, reject) => {
                            if (money === '') {
                                this.$notification.warning({
                                    message: '提示',
                                    description: '请输入数量'
                                })
                                reject()
                            } else {
                                Member.addNodeLog({ uid,money }).then(res => {
                                    resolve()
                                    this.getList()
                                }).catch(res => {
                                    reject()
                                })
                            }
                        })
                    }
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
    .radioTitle{
        display: block;
        font-size: 18px;
        padding: 20px 0px;
    }
    .radioStyle{
        display: block;
        height:30px;
        line-height: 30px;
        margin-left: 20px;
    }
</style>