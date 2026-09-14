<template>
    <PageView>
        <a-spin :spinning="loading">
            <a-card :title="`${form.id?'编辑':'添加'}轮播`">
                <div class="formCon">
                    <a>内容:</a>
                    <tinymceForm v-model="form.content"></tinymceForm>
                </div>
                <div class="formCon">
                    <a>是否显示:</a>
                    <a-switch v-model="form.status"/>
                </div>
                <div class="formCon">
                    <a>标题:</a>
                    <a-input v-model="form.title" placeholder="请输入标题"/>
                </div>
                <a-row :gutter="10" class="inputGroup" style="margin-bottom: 20px" type="flex" justify="space-around" align="middle">
                    <a-col :xs="24" :md="16" :lg="16" :xl="10">
                        <div class="imgContaienr">
                            <div class="con">
                                <img v-if="pic!==``" :src="pic"/>
                                <a v-else class="placeText">请选择图片</a>
                            </div>
                        </div>
                    </a-col>
                    <a-col :xs="24" :md="8" :lg="8" :xl="14">
                        <div class="btnCon">
                            <input @change="handleChange" type="file" accept="image/*"/>
                            <a-button-group>
                                <a-button type="primary">选择图片</a-button>
                            </a-button-group>
                        </div>
                    </a-col>
                </a-row>
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
        name: 'bannerEdit',
        components:{tinymceForm},
        data(){
            return{
                loading:this.$route.query.id?true:false,
                pic:"",
                form:{
                    id:this.$route.query.id||undefined,
                    title:"",
                    content:"",
                    image:undefined,
                    status:true
                }
            }
        },
        mounted(){
            if(this.form.id)this.getOne()
        },
        methods:{
            getOne(){
                Art.details({id:this.form.id}).then(res => {
                    this.loading = false
                    this.pic = projectUrl+res.list.image
                    this.form.title = res.list.title
                    this.form.content = res.list.content
                    this.form.status = res.list["status"]===`1`?true:false
                })
            },
            handleChange(e){
                let file = e.target.files[0];
                let reader = new FileReader();
                reader.onload = () => {
                    this.form.image = e.target.files[0];
                    this.pic = reader.result
                    reader.onload = null;
                };
                reader.readAsDataURL(file);
            },
            sub(){
                this.loading = true
                let form = {...this.form}
                this.$set(form,"status",form.status?`1`:`2`)
                let formData = new FormData()
                Object.keys(form).forEach(v=>{
                    if(form[v] !== undefined)formData.append(v,form[v])
                })
                Art[this.form.id?`changeArticle`:`addArticle`](formData).then(res => {
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