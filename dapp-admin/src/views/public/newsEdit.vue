<template>
    <PageView>
        <a-spin :spinning="loading">
            <a-card :title="`${form.id?'编辑':'添加'}公告`">
                <div class="formCon">
                    <a>标题:</a>
                    <a-input v-model="form.title" placeholder="请输入标题"/>
                    <a>内容:</a>
                    <tinymceForm v-model="form.content"></tinymceForm>
                </div>
                <a-button type="primary" style="margin-top: 20px" @click="sub">
                    提交
                </a-button>
            </a-card>
        </a-spin>
    </PageView>
</template>

<script>
    import Art from '../../api/Art'
    import tinymceForm from "../../components/tinymceForm/tinymceForm"
    export default {
        name: 'newsEdit',
        components:{tinymceForm},
        data(){
            return{
                loading:this.$route.query.id?true:false,
                form:{
                    id:this.$route.query.id||undefined,
                    title:"",
                    content:"",
                }
            }
        },
        mounted(){
            if(this.form.id)this.getOne()
        },
        methods:{
            getOne(){
                Art.getArticleDetails({id:this.form.id}).then(res => {
                    this.loading = false
                    this.form.content = res.data.content;
                    this.form.title = res.data.title;
                })
            },
            sub(){
                this.loading = true
                Art.addArticle(this.form).then(res => {
                    this.loading = false
                    this.$router.back()
                }).catch(res=>{
                    this.loading = false
                })
            }
        }
    }
</script>

<style scoped lang="less">
    .inputGroup{
        margin-top: 20px;
    }
    .imgContaienr{
        border-radius: 8px;
        border:1px solid rgba(0,0,0,0.1);
        padding: 15px;
        .con{
            height: 300px;
            width: 100%;
            display: flex;
            align-items: center;
            justify-content: center;
            img{
                display: block;
                max-height: 100%;
                max-width: 100%;
            }
            .placeText{
                font-size: 20px;
            }
        }
    }
    .formCon{
        a{
            display: block;
            padding: 15px 0px;
        }
    }
    .btnCon{
        position: relative;
        overflow: hidden;
        display: inline-block;
        input{
            display: block;
            position: absolute;
            z-index: 1;
            width: 100%;
            height: 100%;
            background: red;
            opacity: 0;
        }
    }
</style>