<template>
    <PageView>
        <a-spin :spinning="loading">
            <a-card title="新增用户组">
                <a-form :form="form" style="width: 350px">
                    <a-form-item :label-col="labelCol" :wrapper-col="wrapperCol">
                        <a-input v-decorator="[`title`, { rules: [{ required: true, message: '请输入名称' }]}]" placeholder="请输入名称"/>
                    </a-form-item>
                </a-form>
                <!--树形菜单-->
                <a-tree
                    checkable
                    :treeData="inputArray"
                    @check="this.onCheck"
                >
                </a-tree>
                <a-button type="primary" @click="sub">
                    提交
                </a-button>
            </a-card>
        </a-spin>
    </PageView>
</template>

<script>
    import Auth from '../../api/Auth'
    export default {
        name: 'addUser',
        data(){
            return{
                form: this.$form.createForm(this),
                loading:true,
                inputArray:[],
                labelCol:{
                    xs:{span:24},
                    sm:{span:12},
                },
                wrapperCol:{
                    xs:{span:24},
                    sm:{span:12},
                },
                idArray:[],
            }
        },
        mounted(){
            this.getConfig()
        },
        methods:{
            getConfig(){
                this.loading = true
                Auth.getAllRules().then(res => {
                    this.inputArray = res.data.map((v1,k1)=>{
                        v1.key = v1.id
                        v1.children = v1.son.map((v2,k2)=>{
                            v2.key = v2.id
                            v2.children = []
                            return v2
                        })
                        return v1
                    })
                    this.loading = false
                })
            },
            onCheck(checkedKeys, info) {
                this.idArray = checkedKeys
            },
            sub(){
                this.form.validateFields((err, values) => {
                    if (!err) {
                        this.loading = true
                        Auth.addGroup({...values,rule:this.idArray.join(",")}).then(res => {
                            this.loading = false
                            this.$router.back()
                        })
                    }
                });
            }
        }
    }
</script>

<style scoped lang="less">

</style>