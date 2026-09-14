import Vue from 'vue'
import axios from 'axios'
import store from '@/store'
import qs from 'qs';
import {
    VueAxios
} from './axios'
import notification from 'ant-design-vue/es/notification'
import {
    ACCESS_TOKEN
} from '@/store/mutation-types'

// 创建 axios 实例
const service = axios.create({
    timeout: 30000 // 请求超时时间
})

const err = (error) => {
    if (error.response) {
        const data = error.response.data
        const token = Vue.ls.get(ACCESS_TOKEN)
        if (error.response.status === 403) {
            notification.error({
                message: 'Forbidden',
                description: data.message
            })
        }
        if (error.response.status === 401 && !(data.result && data.result.isLogin)) {
            notification.error({
                message: 'Unauthorized',
                description: 'Authorization verification failed'
            })
            if (token) {
                store.dispatch('Logout').then(() => {
                    setTimeout(() => {
                        window.location.reload()
                    }, 1500)
                })
            }
        }else{
            notification.error({
                message: '错误',
                description: data.message||"网络错误"
            })
        }
    }
    return Promise.reject(error)
}

// request interceptor
service.interceptors.request.use(config => {
    const token = Vue.ls.get(ACCESS_TOKEN) || localStorage.getItem('token')
    console.log('token', token)
    if (token) {
        config.headers['Authorization'] = `Bearer ${token}` // 让每个请求携带自定义 token 请根据实际情况自行修改
    }
    if(config.method === "post"){
        /* 修改post Content-type*/
        config.headers["Content-type"]= 'application/x-www-form-urlencoded'
        /*判断是否是new FormData类型*/
        if(!(config.data instanceof FormData)){
            config.data = qs.stringify(config.data);
        }
    }
    return config
}, err)

// response interceptor
service.interceptors.response.use((response) => {
    if (response.config.method === "post") {
        notification.success({
            message: '成功提示',
            description: `操作成功`
        })
    }
    return response.data
}, err)

const installer = {
    vm: {},
    install (Vue) {
        Vue.use(VueAxios, service)
    }
}

export {
    installer as VueAxios,
    service as axios
}
