import Vue from 'vue'
import router from './router'
import store from './store'

import NProgress from 'nprogress' // progress bar
import 'nprogress/nprogress.css' // progress bar style
import notification from 'ant-design-vue/es/notification'
import { setDocumentTitle, domTitle } from '@/utils/domUtil'
import { ACCESS_TOKEN } from '@/store/mutation-types'

NProgress.configure({ showSpinner: false }) // NProgress Configuration
if(Vue.ls.get(ACCESS_TOKEN))store.dispatch('GetInfo');
store.dispatch('setInfo')
router.beforeEach((to, from, next) => {
    NProgress.start() // start progress bar
    to.meta && (typeof to.meta.title !== 'undefined' && setDocumentTitle(`${to.meta.title} - ${domTitle}`))
    const hasToken = Vue.ls.get(ACCESS_TOKEN)
    if (hasToken && to.name === 'login') {
        next({ name: 'index' })
    } else if (!hasToken && to.name !== 'login') {
        next({ name: 'login' })
    } else {
        next()
    }
})

router.afterEach(() => {
  NProgress.done() // finish progress bar
})
/**
 * Action 权限指令
 * 指令用法：
 *  - 在需要控制 action 级别权限的组件上使用 v-action:[method] , 如下：
 *    <i-button v-action:add >添加用户</a-button>
 *    <a-button v-action:delete>删除用户</a-button>
 *    <a v-action:edit @click="edit(record)">修改</a>
 *
 *  - 当前用户没有权限时，组件上使用了该指令则会被隐藏
 *  - 当后台权限跟 pro 提供的模式不同时，只需要针对这里的权限过滤进行修改即可
 *
 *  @see https://github.com/sendya/ant-design-pro-vue/pull/53
 */
const action = Vue.directive('action', {
    bind: function (el, binding, vnode) {
        const actionName = binding.arg
        const roles = store.getters.roles
        const elVal = vnode.context.$route.meta.permission
        const permissionId = elVal instanceof String && [elVal] || elVal
        roles.permissions.forEach(p => {
            if (!permissionId.includes(p.permissionId)) {
                return
            }
            if (p.actionList && !p.actionList.includes(actionName)) {
                el.parentNode && el.parentNode.removeChild(el) || (el.style.display = 'none')
            }
        })
    }
})

export {
    action
}

