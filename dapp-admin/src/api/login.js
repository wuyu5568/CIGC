import api from './index'
import { axios } from '@/utils/request'

/**
 * login func
 * parameter: {
 *     username: '',
 *     password: '',
 *     remember_me: true,
 *     captcha: '12345'
 * }
 * @param parameter
 * @returns {*}
 */
//登录
export function login (parameter) {
    return axios({
        url: '/admin_dhb/login',
        method: 'post',
        data: parameter
    })
}

//登录页面发送验证码
export function getSmsCaptcha (parameter) {
    return axios({
        url: api.SendSms,
        method: 'post',
        data: parameter
    })
}

//获取用户信息
export function getInfo () {
    return axios({
        url: '/user/info',
        method: 'get',
        headers: {
            'Content-Type': 'application/json;charset=UTF-8'
        }
    })
}

//登出
export function logout () {
    return axios({
        url: '/auth/logout',
        method: 'post',
        headers: {
            'Content-Type': 'application/json;charset=UTF-8'
        }
    })
}

