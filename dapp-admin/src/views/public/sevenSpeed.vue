<template>
    <PageView>
        <a-spin :spinning="loading">
            <a-card title="官网配置">
                <a-upload @change="changeState" slot="extra" :action="actionUrl" directory :headers="headers" accept="application/pdf">
                    <a-button type="primary"> <a-icon type="upload" />上传白皮书</a-button>
                </a-upload>
                <a-form :form="form">
                    <a-form-item :label="value.name" v-for="(value,key) in inputArray">
                        <a-input v-if="value.key!==`invite_txt`" style="width: 350px;" v-decorator="[value.key, { rules: [{ required: true, message: '请输入' }],initialValue:value.value}]" placeholder="请输入"/>
                        <a-textarea v-else :rows="5" style="width: 350px;" v-decorator="[value.key, { rules: [{ required: true, message: '请输入' }],initialValue:value.value}]" placeholder="请输入"/>
                    </a-form-item>
                </a-form>
                <a-button type="primary" @click="sub">
                    提交
                </a-button>
            </a-card>
        </a-spin>
    </PageView>
</template>

<script>
    import Config from '../../api/Config'
    import Vue from 'vue'
    import { ACCESS_TOKEN } from '../../store/mutation-types'
    import notification from 'ant-design-vue/lib/notification'
    export default {
        name: 'sevenSpeed',
        data(){
            return{
                form: this.$form.createForm(this),
                loading:true,
                inputArray:[],
                actionUrl:process.env.NODE_ENV === 'production' ? projectUrl+`/Admin/Config/uploadWhitePaper` : `/api/Admin/Config/uploadWhitePaper`, // api base_url
                headers:{
                    token:Vue.ls.get(ACCESS_TOKEN)
                }
            }
        },
        activated(){
            this.getConfig()
        },
        methods:{
            getConfig(){
                this.loading = true
                Config.getWebsiteConfig().then(res => {
                    this.inputArray = res.config
                    this.loading = false
                })
            },
            sub(){
                this.form.validateFields((err, values) => {
                    if (!err) {
                        this.loading = true
                        Config.changeWebsiteConfig ({...values }).then(res => {
                            this.loading = false
                        }).catch(res=>{
                            this.getConfig()
                        })
                    }
                });
            },
            changeState(e){
                if(e.file.response&&e.file.response.status === 1){
                    this.$notification.success({
                        message: '成功提示',
                        description: `上传成功`
                    })
                }
            }
        }
    }
</script>

<style scoped lang="less">

</style>