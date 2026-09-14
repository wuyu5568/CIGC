import { createI18n } from 'vue-i18n';
import zh from '../i18n/language/zh.json'
import en from '../i18n/language/en.json'

export default createI18n({
    legacy: false,
    locale: localStorage.getItem("lan") || "zh",
    fallbackLocale: 'zh',
    missingWarn: false,
    fallbackWarn: false,
    messages: { zh, en },
})