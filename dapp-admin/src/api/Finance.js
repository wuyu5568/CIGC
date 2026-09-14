import { axios } from '@/utils/request'
const api = "/Finance"
export default {
    getFinanceAll:(parameter)=>{
        return axios({
            url: `${api}/getFinanceAll`,
            method: 'post',
            data: parameter
        })
    },
    getAllType:(parameter)=>{
        return axios({
            url: `${api}/getAllType`,
            method: 'post',
            data: parameter
        })
    },
}