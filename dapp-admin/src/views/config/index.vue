<template>
    <PageView>
        <a-spin :spinning="loading">
            <a-card title="配置项参数">
                <a-form :form="form">
                    <a-form-item :label="value.name" v-for="(value,key) in inputArray">
                        <a-textarea v-if="value.key===`invite_txt`||value.key===`invite_txt_en`" :rows="6" style="width: 350px;" v-decorator="[value.key, { rules: [{ required: true, message: '请输入' }],initialValue:value.value}]" placeholder="请输入"/>
                        <a-input v-else  style="width: 350px;" v-decorator="[value.key, { rules: [{ required: true, message: '请输入' }],initialValue:value.value}]" placeholder="请输入"/>
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
    export default {
        name: 'index',
        data(){
            return{
                form: this.$form.createForm(this),
                loading:true,
                inputArray:[]
            }
        },
        activated(){
            this.getConfig()
        },
        methods:{
            getConfig(){
                this.loading = true
                Config.getAllConfig().then(res => {
                    this.inputArray = res.config
                    this.loading = false
                })
            },
            sub(){
                this.form.validateFields((err, values) => {
                    if (!err) {
                        this.loading = true
                        Config.changeConfig ({...values }).then(res => {
                            this.loading = false
                        }).catch(res=>{
                            this.getConfig()
                        })
                    }
                });
            }
        }
    }
</script>

<style scoped lang="less">

</style>