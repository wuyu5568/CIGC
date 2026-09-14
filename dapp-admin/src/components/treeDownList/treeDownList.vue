<template>
    <div class="container">
        <div class="con" v-for="(value, key) in list">
            <div class="dataCon" :style="{ width: `calc(100% - ${index * 20}px)` }">
                <div class="data" @click="look(value)">
                    <a>{{ value.address }}</a>
                    <a>{{ value.createdAt }}</a>
                </div>
                <div class="circle" @click="look(value)">
                    {{ value.child.length === 0 ? `+` : `-` }}
                </div>
            </div>
            <treeDownList :list="value.child" :index="index + 1"></treeDownList>
        </div>
    </div>
</template>

<script>
import Gai from '../../api/Gai'
export default {
    props: {
        list: {
            type: Array,
            default: () => {
                return []
            }
        },
        index: {
            type: Number,
            default: 0,
        }
    },
    methods: {
        look(value) {
            if (value.child.length !== 0) {
                value.child = []
            } else {
                Gai.user_recommend({ userId: value.userId }).then(res => {
                    value.child = res.users.map(v => {
                        v.child = []
                        return v
                    })
                    if (value.child.length === 0) {
                        this.$notification.error({
                            message: '提示',
                            description: `暂无下级`
                        })
                    }
                })
            }
        }
    }
}
</script>

<style scoped lang="less">
@import "treeDownList";
</style>