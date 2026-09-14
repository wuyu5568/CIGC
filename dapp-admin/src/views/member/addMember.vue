<template>
    <PageView>
        <a-spin :spinning="loading">
            <a-card :title="uid?`编辑会员`:`新增会员`">
                <a-form :form="form" style="width: 350px">
                    <a-form-item v-if="uid" label="动态等级" :label-col="labelCol" :wrapper-col="wrapperCol">
                        <a-select v-decorator="['level',{rules: [{ required: true, message: '请选选择等级' }] }]" placeholder="请选选择等级">
                            <a-select-option :value="Number(i-1)" v-for="i in 6" :key="i">
                                {{i-1}}
                            </a-select-option>
                        </a-select>
                    </a-form-item>
                </a-form>
                <a-button type="primary" @click="sub">
                    {{uid?"修改":"新增"}}
                </a-button>
            </a-card>
        </a-spin>
    </PageView>
</template>

<script type="text/jsx">
    import Member from '../../api/Member'
    export default {
        name: 'addMember',
        data(){
            return{
                form: null,
                loading:true,
                uid:undefined,
                labelCol:{
                    xs:{span:24},
                    sm:{span:8},
                },
                wrapperCol:{
                    xs:{span:24},
                    sm:{span:16},
                },
            }
        },
        activated(){
            this.$nextTick(()=>{
                this.form = this.$form.createForm(this)
                this.form.resetFields()
                this.uid = this.$route.query.uid||undefined
                this.uid?this.getPerson():this.loading = false
            })
        },
        methods:{
            getPerson(){
                Member.getUserDetails({uid:this.uid}).then(res => {
                    this.$nextTick(()=>{
                        this.form.setFieldsValue({level: parseInt(res.data.level)})
                        this.loading = false
                    })
                })
            },
            sub(){
                this.form.validateFields((err, values) => {
                    if (!err) {
                        this.loading = true
                        Member[this.uid?`updateUser`:`addMember`]({uid:this.uid,...values }).then(res => {
                            this.loading = false
                            this.$router.back()
                        }).catch(res=>{
                            this.loading = false
                        })
                    }
                });
            }
        }
    }
</script>

<style scoped lang="less">

</style>