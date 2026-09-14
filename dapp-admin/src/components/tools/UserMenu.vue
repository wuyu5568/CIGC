<template>
    <div class="user-wrapper">
        <a @click="handleChange" v-if="showFullScreenBtn">
            <span class="action">
                <a-icon :type="!isFull?`fullscreen`:`fullscreen-exit`"/>
            </span>
        </a>
        <!--<notice-icon class="action"/>-->
        <a-dropdown>
        <span class="action ant-dropdown-link user-dropdown-menu">
            <a-avatar class="avatar" size="small" style="backgroundColor:#87d068;color:#fff" icon="user"/>
            <span>{{ nickname() }}</span>
        </span>
            <a-menu slot="overlay" class="user-dropdown-menu-wrapper">
                <!--<a-menu-item key="1">
                    <router-link :to="{ name: 'settings' }">
                        <a-icon type="setting"/>
                        <span>账户设置</span>
                    </router-link>
                </a-menu-item>-->
                <a-menu-item key="3">
                    <a href="javascript:;" @click="handleLogout">
                        <a-icon type="logout"/>
                        <span>退出登录</span>
                    </a>
                </a-menu-item>
            </a-menu>
        </a-dropdown>
    </div>
</template>

<script>
    import NoticeIcon from '@/components/NoticeIcon'
    import { mapActions, mapGetters } from 'vuex'

    export default {
        name: 'UserMenu',
        data(){
            return{
                isFull:false,
            }
        },
        computed: {
            showFullScreenBtn () {
                return window.navigator.userAgent.indexOf('MSIE') < 0;
            }
        },
        components: {
            NoticeIcon
        },
        created () {
            let isFullscreen = document.fullscreenElement || document.mozFullScreenElement || document.webkitFullscreenElement || document.fullScreen || document.mozFullScreen || document.webkitIsFullScreen;
            isFullscreen = !!isFullscreen;
            document.addEventListener('fullscreenchange', () => {
                this.isFull = !this.isFull
            });
            document.addEventListener('mozfullscreenchange', () => {
                this.isFull = !this.isFull
            });
            document.addEventListener('webkitfullscreenchange', () => {
                this.isFull = !this.isFull
            });
            document.addEventListener('msfullscreenchange', () => {
                this.isFull = !this.isFull
            });
        },
        methods: {
            ...mapActions(['Logout']),
            ...mapGetters(['nickname', 'avatar']),
            handleFullscreen () {
                let main = document.body;
                if (this.isFull) {
                    if (document.exitFullscreen) {
                        document.exitFullscreen();
                    } else if (document.mozCancelFullScreen) {
                        document.mozCancelFullScreen();
                    } else if (document.webkitCancelFullScreen) {
                        document.webkitCancelFullScreen();
                    } else if (document.msExitFullscreen) {
                        document.msExitFullscreen();
                    }
                } else {
                    if (main.requestFullscreen) {
                        main.requestFullscreen();
                    } else if (main.mozRequestFullScreen) {
                        main.mozRequestFullScreen();
                    } else if (main.webkitRequestFullScreen) {
                        main.webkitRequestFullScreen();
                    } else if (main.msRequestFullscreen) {
                        main.msRequestFullscreen();
                    }
                }
            },
            handleChange () {
                this.handleFullscreen();
            },
            handleLogout () {
                const that = this

                this.$confirm({
                    title: '提示',
                    content: '真的要注销登录吗 ?',
                    onOk () {
                        return that.Logout({}).then(() => {
                            window.location.reload()
                        }).catch(err => {
                            that.$message.error({
                                title: '错误',
                                description: err.message
                            })
                        })
                    },
                    onCancel () {
                    }
                })
            }
        },
    }
</script>
