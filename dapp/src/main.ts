import "lib-flexible/flexible.js";
import { createApp } from 'vue';
import 'vant/lib/index.css';
import "./style/index.less";
import i18n from "./language";
import vue3Clipboard from 'vue3.0-clipboard';
import directive from "./directive";
import pinia from './pinia'
import App from './App.vue';
import Antd from 'ant-design-vue';
import regComponents from "./components/index";
import Vant, { Locale } from 'vant';
import zhCN from 'vant/es/locale/lang/zh-CN';
import enUS from 'vant/es/locale/lang/en-US';
import router from './router'

Locale.use((localStorage.getItem('lan') || 'zh') === 'en' ? 'en-US' : 'zh-CN', (localStorage.getItem('lan') || 'zh') === 'en' ? enUS : zhCN)

const app = createApp(App);
app.use(i18n);
app.use(pinia);
app.use(vue3Clipboard);
app.use(Vant);
regComponents(app);
directive(app);
app.use(Antd);
app.use(router);
app.mount('#app')