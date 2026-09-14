import Vue from 'vue'
import { login, getInfo, logout } from '@/api/login'
import { ACCESS_TOKEN, ACCESS_NAME, ACCESS_AVATAR } from '@/store/mutation-types'
import { welcome } from '@/utils/util'
import { asyncRouterMap } from '@/config/myRouter'
import router from '@/router'
import Gai from "@/api/Gai";
const user = {
    state: {
        token: '',
        name: '',
        welcome: '',
        avatar: '',
        roles: [],
        info: {},
        routerList: [],
    },

    mutations: {
        SET_TOKEN: (state, token) => {
            state.token = token
        },
        SET_NAME: (state, { name, welcome }) => {
            state.name = name
            state.welcome = welcome
        },
        SET_AVATAR: (state, avatar) => {
            state.avatar = avatar
        },
        SET_ROLES: (state, roles) => {
            state.roles = roles
        },
        SET_INFO: (state, info) => {
            state.info = info
        }
    },

    actions: {
        // 登录
        Login({ commit }, userInfo) {
            return new Promise((resolve, reject) => {
                Gai.login(userInfo).then(response => {
                    Vue.ls.set(ACCESS_TOKEN, response.token, 2 * 24 * 60 * 60 * 1000)
                    Vue.ls.set(ACCESS_NAME, "管理员", 2 * 24 * 60 * 60 * 1000)
                    // Vue.ls.set(ACCESS_AVATAR, response.data.admin.photo, 2 * 24 * 60 * 60 * 1000)
                    commit('SET_TOKEN', response.token)
                    commit('SET_NAME', { name: "管理员", welcome: welcome() })
                    // commit('SET_AVATAR', response.data.admin.photo)
                    resolve()
                }).catch(error => {
                    reject(error)
                })
            })
        },
        // 获取用户信息
        GetInfo({ commit, state }) {
            return new Promise((resolve, reject) => {
                Gai.my_auth_list().then(res => {
                    const baseList = asyncRouterMap;
                    const auth = res.auth.map(v => v.path);
                    if (res.super !== `1`) {
                        /* 普通管理 */
                        baseList[0].children = baseList[0].children.filter(v => auth.includes(v.path));
                    }
                    state.routerList = baseList;
                    router.addRoutes(state.routerList);
                    resolve(state.routerList[0].children[0].name);
                }).catch(error => {
                    reject(error)
                })
            })
        },

        // 登出
        Logout({ commit, state }) {
            return new Promise((resolve) => {
                commit('SET_TOKEN', '')
                commit('SET_ROLES', [])
                Vue.ls.remove(ACCESS_TOKEN)

                logout(state.token).then(() => {
                    resolve()
                }).catch(() => {
                    resolve()
                })
            })
        },
        setInfo({ commit }) {
            return new Promise((resolve) => {
                if (Vue.ls.get(ACCESS_TOKEN)) {
                    commit('SET_NAME', { name: Vue.ls.get(ACCESS_NAME), welcome: welcome() })
                    commit('SET_AVATAR', Vue.ls.get(ACCESS_AVATAR))
                }
                resolve()
            })
        }
    }
}

export default user
