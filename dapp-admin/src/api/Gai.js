import { axios } from '@/utils/request'
const api8005 = `${projectUrl}/api/admin_dhb`
const api8006 = `${projectUrl}/api/admin_dhb`
export default {
    sub_money: (parameter) => {
        return axios({
            url: `${api8005}/sub_money`,
            method: 'get',
            params: parameter
        })
    },
    reward_list: (parameter) => {
        return axios({
            url: `${api8005}/reward_list`,
            method: 'get',
            params: parameter
        })
    },
    dividend_policy: (parameter) => {
        return axios({
            url: `${api8005}/lock_user_reward`,
            method: 'post',
            data: parameter
        })
    },
    buy_list: (parameter) => {
        return axios({
            url: `${api8005}/buy_list`,
            method: 'get',
            params: parameter
        })
    },
    buy_list_2: (parameter) => {
        return axios({
            url: `${api8005}/buy_list`,
            method: 'get',
            params: parameter
        })
    },
    buy_list_3: (parameter) => {
        return axios({
            url: `${api8005}/buy_list`,
            method: 'get',
            params: parameter
        })
    },
    buy_list_4: (parameter) => {
        return axios({
            url: `${api8005}/buy_list`,
            method: 'get',
            params: parameter
        })
    },
    order_pay: (parameter) => {
        return axios({
            url: `${api8005}/order_pay`,
            method: 'post',
            data: parameter
        })
    },
    deposit_scan: (parameter) => {
        return axios({
            url: `${api8005}/deposit_scan`,
            method: 'post',
            data: parameter || {}
        })
    },
    settle: (parameter) => {
        return axios({
            url: `${api8005}/settle`,
            method: 'post',
            data: parameter || {}
        })
    },
    settle_status: (parameter) => {
        return axios({
            url: `${api8005}/settle_status`,
            method: 'get',
            params: parameter
        })
    },
    settle_reset: (parameter) => {
        return axios({
            url: `${api8005}/settle_reset`,
            method: 'post',
            data: parameter || {}
        })
    },
    recommend_list: (parameter) => {
        return axios({
            url: `${api8005}/recommend_list`,
            method: 'get',
            params: parameter
        })
    },
    placement: (parameter) => {
        return axios({
            url: `${api8005}/placement`,
            method: parameter && parameter._method === 'get' ? 'get' : 'post',
            params: parameter && parameter._method === 'get' ? parameter : undefined,
            data: parameter && parameter._method === 'get' ? undefined : parameter
        })
    },
    order_buy_four: (parameter) => {
        return axios({
            url: `${api8005}/set_buy_four`,
            method: 'post',
            data: parameter
        })
    },
    // 手动补差价：参数 id(uint64)、address(string)、amountOne/amountTwo/amountThree(double)
    buy_four_diff: (parameter) => {
        return axios({
            url: `${api8005}/update_buy_four`,
            method: 'post',
            data: parameter
        })
    },
    user_list: (parameter) => {
        return axios({
            url: `${api8005}/user_list`,
            method: 'get',
            params: parameter
        })
    },
    location_list: (parameter) => {
        return axios({
            url: `${api8005}/location_list`,
            method: 'get',
            params: parameter
        })
    },
    location_list_2: (parameter) => {
        return axios({
            url: `${api8005}/location_list_2`,
            method: 'get',
            params: parameter
        })
    },
    withdraw_list: (parameter) => {
        return axios({
            url: `${api8005}/withdraw_list`,
            method: 'get',
            params: parameter
        })
    },
    withdraw_pass: (parameter) => {
        return axios({
            url: `${api8005}/withdraw_pass`,
            method: 'post',
            data: parameter
        })
    },
    withdraw_reject: (parameter) => {
        return axios({
            url: `${api8005}/withdraw_reject`,
            method: 'post',
            data: parameter
        })
    },
    withdraw_payout: (parameter) => {
        return axios({
            url: `${api8005}/withdraw_payout`,
            method: 'post',
            data: parameter || {}
        })
    },
    adjust_balance: (parameter) => {
        return axios({
            url: `${api8005}/adjust_balance`,
            method: 'post',
            data: parameter
        })
    },
    all: (parameter) => {
        return axios({
            url: `${api8005}/all`,
            method: 'get',
            params: parameter
        })
    },
    month_recommend: (parameter) => {
        return axios({
            url: `${api8005}/month_recommend`,
            method: 'get',
            params: parameter
        })
    },
    config: (parameter) => {
        return axios({
            url: `${api8005}/config`,
            method: 'get',
            params: parameter
        })
    },
    config_update: (parameter) => {
        return axios({
            url: `${api8005}/config_update`,
            method: 'post',
            data: parameter
        })
    },
    downline: (parameter) => {
        return axios({
            url: `${api8005}/downline`,
            method: 'get',
            params: parameter
        })
    },
    user_recommend: (parameter) => {
        const params = { ...(parameter || {}) }
        if (!params.address && params.userId) {
            params.address = params.userId
        }
        return axios({
            url: `${api8005}/recommend_list`,
            method: 'get',
            params
        }).then((res) => {
            const rows = (res && (res.recommends || res.users)) || []
            return { ...res, users: rows, recommends: rows }
        })
    },
    record_list: (parameter) => {
        return axios({
            url: `${api8005}/record_list`,
            method: 'get',
            params: parameter
        }).then((res) => {
            const rows = (res && (res.locations || res.rewards)) || []
            return { ...res, locations: rows, count: res.count }
        })
    },
    trade_list: (parameter) => {
        return axios({
            url: `${projectUrl}/api/app_server/package_list`,
            method: 'get',
            params: parameter
        }).then((res) => {
            const items = (res && (res.items || res.goods)) || []
            return { ...res, goods: items, count: String(items.length) }
        })
    },
    trade_list_2: (parameter) => {
        return axios({
            url: `${projectUrl}/api/app_server/package_list`,
            method: 'get',
            params: parameter
        }).then((res) => {
            const items = (res && (res.items || res.goods)) || []
            return { ...res, goods: items, count: String(items.length) }
        })
    },
    trade_list_3: (parameter) => {
        return axios({
            url: `${api8005}/web3_goods`,
            method: 'get',
            params: parameter
        })
    },
    web3_goods_list: (parameter) => {
        return axios({
            url: `${api8005}/web3_goods`,
            method: 'get',
            params: parameter
        })
    },
    web3_goods_create: (parameter) => {
        return axios({
            url: `${api8005}/web3_goods_create`,
            method: 'post',
            data: parameter
        })
    },
    web3_goods_update: (parameter) => {
        return axios({
            url: `${api8005}/web3_goods_update`,
            method: 'post',
            data: parameter
        })
    },
    web3_goods_status: (parameter) => {
        return axios({
            url: `${api8005}/web3_goods_status`,
            method: 'post',
            data: parameter
        })
    },
    web3_goods_delete: (parameter) => {
        return axios({
            url: `${api8005}/web3_goods_delete`,
            method: 'post',
            data: parameter
        })
    },
    web3_goods_image_upload: (parameter) => {
        return axios({
            url: `${api8005}/web3_goods_image_upload`,
            method: 'post',
            data: parameter,
            notify: false
        })
    },
    web3_goods_detail: (parameter) => {
        return axios({
            url: `${api8005}/web3_goods_detail`,
            method: 'get',
            params: parameter
        })
    },
    package_create: (parameter) => {
        return axios({
            url: `${api8005}/package_create`,
            method: 'post',
            data: parameter
        })
    },
    package_update: (parameter) => {
        return axios({
            url: `${api8005}/package_update`,
            method: 'post',
            data: parameter
        })
    },
    package_delete: (parameter) => {
        return axios({
            url: `${api8005}/package_delete`,
            method: 'post',
            data: parameter
        })
    },
    /* 登录相关 */
    login: (parameter) => {
        return axios({
            url: `${api8006}/login`,
            method: 'post',
            data: parameter
        })
    },
    admin_list: (parameter) => {
        return axios({
            url: `${api8006}/admin_list`,
            method: 'get',
            params: parameter
        })
    },
    change_password: (parameter) => {
        return axios({
            url: `${api8006}/change_password`,
            method: 'post',
            data: parameter
        })
    },
    create_account: (parameter) => {
        return axios({
            url: `${api8006}/create_account`,
            method: 'post',
            data: parameter
        })
    },
    auth_list: (parameter) => {
        return axios({
            url: `${api8006}/auth_list`,
            method: 'get',
            params: parameter
        })
    },
    user_auth_list: (parameter) => {
        return axios({
            url: `${api8006}/user_auth_list`,
            method: 'get',
            params: parameter
        })
    },
    auth_create: (parameter) => {
        return axios({
            url: `${api8006}/auth_create`,
            method: 'post',
            data: parameter
        })
    },
    auth_delete: (parameter) => {
        return axios({
            url: `${api8006}/auth_delete`,
            method: 'post',
            data: parameter
        })
    },
    my_auth_list: (parameter) => {
        return axios({
            url: `${api8006}/my_auth_list`,
            method: 'get',
            params: parameter
        })
    },
    vip_update: (parameter) => {
        return axios({
            url: `${api8006}/vip_update`,
            method: 'post',
            data: parameter
        })
    },
    change_address: (parameter) => {
        return axios({
            url: `${api8006}/change_address`,
            method: 'post',
            data: parameter
        })
    },
    balance_update: (parameter) => {
        return axios({
            url: `${api8005}/amount_four_update`,
            method: 'post',
            data: parameter
        })
    },
    principal_update: (parameter) => {
        return axios({
            url: `${api8005}/add_money_two`,
            method: 'post',
            data: parameter
        })
    },
    set_isPay: (parameter) => {
        return axios({
            url: `${api8005}/set_ispay`,
            method: 'post',
            data: parameter
        })
    },
    add_lock: (parameter) => {
        return axios({
            url: `${api8005}/add_lock`,
            method: 'post',
            data: parameter
        })
    },
    add_lock_ispay: (parameter) => {
        return axios({
            url: `${api8005}/add_lock_ispay`,
            method: 'post',
            data: parameter
        })
    },
    node_update: (parameter) => {
        return axios({
            url: `${api8005}/add_money_three`,
            method: 'post',
            data: parameter
        })
    },
    set_pass: (parameter) => {
        return axios({
            url: `${api8005}/set_pass`,
            method: 'post',
            data: parameter
        })
    },
    admin_update_location_new_max: (parameter) => {
        return axios({
            url: `${api8005}/admin_update_location_new_max`,
            method: 'post',
            data: parameter
        })
    },
    password_update: (parameter) => {
        return axios({
            url: `${api8005}/password_update`,
            method: 'post',
            data: parameter
        })
    },
    location_insert: (parameter) => {
        return axios({
            url: `${api8005}/location_insert`,
            method: 'post',
            data: parameter
        })
    },
    undo_lock: (parameter) => {
        return axios({
            url: `${api8006}/lock_user`,
            method: 'post',
            data: parameter
        })
    },
    recommend_level: (parameter) => {
        return axios({
            url: `${api8006}/admin_recommend_level`,
            method: 'post',
            data: parameter
        })
    },
    undo_update: (parameter) => {
        return axios({
            url: `${api8006}/undo_update`,
            method: 'post',
            data: parameter
        })
    },
    level_update: (parameter) => {
        return axios({
            url: `${api8006}/level_update`,
            method: 'post',
            data: parameter
        })
    },
    vip_delete: (parameter) => {
        return axios({
            url: `${api8006}/vip_delete`,
            method: 'post',
            data: parameter
        })
    },
    product_status: (parameter) => {
        return axios({
            url: `${api8006}/update_goods`,
            method: 'post',
            data: parameter
        })
    },
    product_status_2: (parameter) => {
        return axios({
            url: `${api8006}/update_goods_two`,
            method: 'post',
            data: parameter
        })
    },
    product_status_3: (parameter) => {
        return axios({
            url: `${api8006}/update_goods_three`,
            method: 'post',
            data: parameter
        })
    },
    product_cover_upload: (parameter) => {
        return axios({
            url: `${api8006}/upload`,
            method: 'post',
            data: parameter
        })
    },
    product_cover_upload_2: (parameter) => {
        return axios({
            url: `${api8006}/upload_two`,
            method: 'post',
            data: parameter
        })
    },
    product_cover_upload_3: (parameter) => {
        return axios({
            url: `${api8006}/upload_three`,
            method: 'post',
            data: parameter
        })
    }
}