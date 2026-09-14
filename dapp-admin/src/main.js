// ie polyfill
import '@babel/polyfill'

import Vue from 'vue'
import App from './App.vue'
import router from './router'
import store from './store/'
import { VueAxios } from './utils/request'

// mock
//import './mock'，模拟数据

import bootstrap from './core/bootstrap'
import './core/use'
import './permission' // permission control
import './utils/filter' // global filter

Vue.config.productionTip = false
Vue.prototype.imgHttpUrl = projectUrl;//项目图片域名

import treeDownList from "./components/treeDownList/treeDownList"
Vue.component("treeDownList",treeDownList)

// mount axios Vue.$http and this.$http
Vue.use(VueAxios)
import { PageView } from '@/layouts'
Vue.component("PageView",PageView)
new Vue({
  router,
  store,
  created () {
    bootstrap()
  },
  render: h => h(App)
}).$mount('#app')
