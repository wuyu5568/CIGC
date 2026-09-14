import { axios } from '@/utils/request'
const api = "/Config"
export default {
    getAllConfig:(parameter)=>{
        return axios({
            url: `${api}/getAllConfig`,
            method: 'post',
            data: parameter
        })
    },
    changeConfig:(parameter)=>{
        return axios({
            url: `${api}/changeConfig`,
            method: 'post',
            data: parameter
        })
    },
    getWebsiteConfig:(parameter)=>{
        return axios({
            url: `${api}/getWebsiteConfig`,
            method: 'post',
            data: parameter
        })
    },
    changeWebsiteConfig:(parameter)=>{
        return axios({
            url: `${api}/changeWebsiteConfig`,
            method: 'post',
            data: parameter
        })
    },
    uploadWhitePaper:(parameter)=>{
        return axios({
            url: `${api}/uploadWhitePaper`,
            method: 'post',
            data: parameter
        })
    },
}