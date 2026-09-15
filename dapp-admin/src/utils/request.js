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

function errorMessage (data, fallback) {
    if (data && typeof data === 'object') {
        return data.message || data.reason || fallback
    }
    if (typeof data === 'string' && data) {
        try {
            const parsed = JSON.parse(data)
            return parsed.message || parsed.reason || fallback
        } catch (e) {
            return data
        }
    }
    return fallback
}

const err = (error) => {
    if (error.response) {
        const data = error.response.data
        const status = error.response.status
        const url = (error.config && error.config.url) || ''
        if (status === 403) {
            notification.error({
                message: '没有权限',
                description: errorMessage(data, '操作被拒绝')
            })
            return Promise.reject(error)
        }
        if (status === 401 && !(data && data.result && data.result.isLogin)) {
            const sessionCheck = /my_auth_list/.test(url)
            const token = Vue.ls.get(ACCESS_TOKEN)
            if (sessionCheck) {
                notification.error({
                    message: '登录已失效',
                    description: errorMessage(data, '请重新登录')
                })
                if (token) {
                    store.dispatch('Logout').then(() => {
                        if (window.location.hash.indexOf('/user/login') === -1) {
                            window.location.hash = '#/user/login'
                        }
                    })
                }
            } else {
                notification.error({
                    message: '请求失败',
                    description: errorMessage(data, '未授权')
                })
            }
        } else {
            notification.error({
                message: '错误',
                description: errorMessage(data, '网络错误')
            })
        }
    }
    return Promise.reject(error)
}

// request interceptor
service.interceptors.request.use(config => {
    const token = Vue.ls.get(ACCESS_TOKEN)
    config.headers = config.headers || {}
    if (token) {
        const bearer = `Bearer ${token}`
        config.headers.common = config.headers.common || {}
        config.headers.common.Authorization = bearer
        config.headers.Authorization = bearer
        if (config.method === 'post') {
            config.headers.post = config.headers.post || {}
            if (typeof config.headers.post === 'object') {
                config.headers.post.Authorization = bearer
            }
        }
    }
    if (config.method === 'post') {
        if (config.data instanceof FormData) {
            delete config.headers['Content-type']
            delete config.headers['Content-Type']
        } else {
            config.headers['Content-type'] = 'application/x-www-form-urlencoded'
            config.data = qs.stringify(config.data)
        }
    }
    return config
}, err)

// response interceptor
service.interceptors.response.use((response) => {
    if (response.config.method === "post" && response.config.notify !== false) {
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
