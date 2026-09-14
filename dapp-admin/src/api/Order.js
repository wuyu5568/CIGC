import { axios } from '@/utils/request'
const api = "/Order"
export default {
    getAllOrder:(parameter)=>{
        return axios({
            url: `${api}/getAllOrder`,
            method: 'post',
            data: parameter
        })
    },
    orderDetalis:(parameter)=>{
        return axios({
            url: `${api}/orderDetalis`,
            method: 'post',
            data: parameter
        })
    },
    appealAdopt:(parameter)=>{
        return axios({
            url: `${api}/appealAdopt`,
            method: 'post',
            data: parameter
        })
    },
    appealRefuse:(parameter)=>{
        return axios({
            url: `${api}/appealRefuse`,
            method: 'post',
            data: parameter
        })
    },
}