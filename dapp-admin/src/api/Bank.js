import { axios } from '@/utils/request'
const api = "/Bank"
export default {
    bankList:(parameter)=>{
        return axios({
            url: `${api}/bankList`,
            method: 'post',
            data: parameter
        })
    },
    start:(parameter)=>{
        return axios({
            url: `${api}/start`,
            method: 'post',
            data: parameter
        })
    },
    getBanklog:(parameter)=>{
        return axios({
            url: `${api}/getBanklog`,
            method: 'post',
            data: parameter
        })
    },
    productList:(parameter)=>{
        return axios({
            url: `${api}/productList`,
            method: 'post',
            data: parameter
        })
    },
    addProduct:(parameter)=>{
        return axios({
            url: `${api}/addProduct`,
            method: 'post',
            data: parameter
        })
    },
    getProductlog:(parameter)=>{
        return axios({
            url: `${api}/getProductlog`,
            method: 'post',
            data: parameter
        })
    },
    getProfitlog:(parameter)=>{
        return axios({
            url: `${api}/getProfitlog`,
            method: 'post',
            data: parameter
        })
    },
    getBankList:(parameter)=>{
        return axios({
            url: `${api}/getBankList`,
            method: 'post',
            data: parameter
        })
    },
    updateBankStatus:(parameter)=>{
        return axios({
            url: `${api}/updateBankStatus`,
            method: 'post',
            data: parameter
        })
    },
}