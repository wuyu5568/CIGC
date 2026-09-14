import { axios } from '@/utils/request'
const api = "/Log"
export default {
    getRechargeLog:(parameter)=>{
        return axios({
            url: `${api}/getRechargeLog`,
            method: 'post',
            data: parameter
        })
    },
    getWithdrawLog:(parameter)=>{
        return axios({
            url: `${api}/getWithdrawLog`,
            method: 'post',
            data: parameter
        })
    },
    getExchangelog:(parameter)=>{
        return axios({
            url: `${api}/getExchangelog`,
            method: 'post',
            data: parameter
        })
    },
    getOrelog:(parameter)=>{
        return axios({
            url: `${api}/getOrelog`,
            method: 'post',
            data: parameter
        })
    },
    getCZlog:(parameter)=>{
        return axios({
            url: `${api}/getCZlog`,
            method: 'post',
            data: parameter
        })
    },
    collect:(parameter)=>{
        return axios({
            url: `${api}/collect`,
            method: 'post',
            data: parameter
        })
    },
    recharge_adopt:(parameter)=>{
        return axios({
            url: `${api}/recharge_adopt`,
            method: 'post',
            data: parameter
        })
    },
    recharge_refuse:(parameter)=>{
        return axios({
            url: `${api}/recharge_refuse`,
            method: 'post',
            data: parameter
        })
    },
    getNodeLog:(parameter)=>{
        return axios({
            url: `${api}/getNodeLog`,
            method: 'post',
            data: parameter
        })
    },
    withdraw_adopt:(parameter)=>{
        return axios({
            url: `${api}/withdraw_adopt`,
            method: 'post',
            data: parameter
        })
    },
    withdraw_refuse:(parameter)=>{
        return axios({
            url: `${api}/withdraw_refuse`,
            method: 'post',
            data: parameter
        })
    },
    getTransferLog:(parameter)=>{
        return axios({
            url: `${api}/getTransferLog`,
            method: 'post',
            data: parameter
        })
    },
}