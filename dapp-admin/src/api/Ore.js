import { axios } from '@/utils/request'
const api = "/Ore"
export default {
    oreList:(parameter)=>{
        return axios({
            url: `${api}/oreList`,
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
    getOrelog:(parameter)=>{
        return axios({
            url: `${api}/getOrelog`,
            method: 'post',
            data: parameter
        })
    },
}