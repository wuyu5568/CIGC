import axios from "axios";
import { showLoadingToast, showFailToast, showSuccessToast, closeToast } from "vant";
import userPerson from "../pinia/person";
import lang from '@/i18n/index'

const instance = axios.create({
    baseURL: import.meta.env.DEV ? import.meta.env.VITE_API + "/api/" : import.meta.env.VITE_API + "/api/",
});
// 请求拦截
instance.interceptors.request.use((config: any) => {
    let token = localStorage.getItem("token")
    if (token) {
        config.headers['Authorization'] = `Bearer ${token}`;
    }
    /* 加载转圈 */
    if (config.data?.loading || config.params?.loading) {
        config.data?.loading ? delete config.data.loading : delete config.params.loading;
        showLoadingToast({
            message: lang("加载中"), duration: 0, overlay: true, overlayStyle: {
                background: "transparent"
            }
        });
    }
    return config;
}, err => {
    showFailToast(err.response?.status);
    return Promise.reject(err);
})
// 响应拦截
instance.interceptors.response.use(res => {
    closeToast();
    if (res.config.method === "post") {
        // 操作失败
        if (res.data?.status === "fail") {
            showFailToast(lang("操作失败"));
            return Promise.reject(lang("操作失败"));
        } else {
            if (res.config?.data && !(JSON.parse(res.config?.data)?.nosuccess) && res.config?.data.status === 'OK') {
                showSuccessToast(lang("操作成功"))
            }
        }
    }
    return res.data;
}, err => {
    closeToast();
    /* 报错提示 */
    const message = err.response?.data?.message;
    const reason = err.response?.data?.reason;
    const status = err.response.status;
    const noMsg = err.config.method === "post" && err.config?.data && JSON.parse(err.config.data)?.noMsg;
    const language = localStorage.getItem("language") || "zh";

    if (!noMsg) {
        if (message) {
            showFailToast(language === "zh" ? message : reason);
        } else {
            showFailToast(err.response?.statusText);
        }
    }
    /* 自动退出登录 */
    if (status === 401 || message === "user not found") {
        showFailToast(lang("登录过期"));
        const person = userPerson();
        person.outLogin();
    }
    return Promise.reject(err);
})
export default instance;