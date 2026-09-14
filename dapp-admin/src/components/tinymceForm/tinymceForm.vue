<template>
    <Editor id="tinymce" v-model="tinymceHtml" :init="editorInit" />
</template>

<script>
    import tinymce from 'tinymce/tinymce'
    import Editor from '@tinymce/tinymce-vue'
    // 引入富文本编辑器主题的js和css
    import 'tinymce/themes/silver/theme'
    import "tinymce/icons/default/icons.min.js"
    // 扩展插件
    import 'tinymce/plugins/image'
    import 'tinymce/plugins/link'
    import 'tinymce/plugins/code'
    import 'tinymce/plugins/table'
    import 'tinymce/plugins/lists'
    import 'tinymce/plugins/wordcount'
    export default {
        name: "tinymceForm",
        components: { Editor },
        props:{
            value:{
                type:String,
                default:""
            },
            height:{
                type:Number,
                default:500
            }
        },
        data(){
            return{
                tinymceHtml:this.value,
            }
        },
        computed:{
            editorInit(){
                return{
                    selector: '#tinymce',
                    language_url: '/tinymce/langs/zh_CN.js',
                    language: 'zh_CN',
                    skin_url: '/tinymce/skins/ui/oxide',
                    height: this.height,
                    mobile: {
                        menubar: true,
                        toolbar_drawer:true,
                    },
                    plugins: 'link lists image code table wordcount',
                    toolbar: 'undo redo | bold italic underline strikethrough | fontsizeselect | forecolor backcolor | alignleft aligncenter alignright alignjustify | bullist numlist | outdent indent blockquote | link unlink ',/*image*/
                    images_upload_handler: (blobInfo, success, failure) => {
                        this.handleImgUpload(blobInfo, success, failure)
                    },
                    statusbar: true, // 底部的状态栏
                    menubar: true, // 最上方的菜单
                    branding: false, // 水印“Powered by TinyMCE”
                    toolbar_drawer:true,
                }
            }
        },
        watch:{
            value(v){
                this.tinymceHtml = v
            },
            tinymceHtml(v){
                this.$emit("input",v)
            }
        },
        mounted() {
            tinymce.init({})
        },
        methods: {
            // 图片上传
            handleImgUpload(blobInfo, success, failure) {
                let formData = new FormData()
                formData.append("image",blobInfo.blob())
                User.uploadPic1(formData).then(res => {
                     success(`${projectUrl}${res.data}`)
                }).catch(() => { failure('error') })
            }
        },
    }
</script>

<style scoped lang="less">
    @import "tinymceForm";
</style>
