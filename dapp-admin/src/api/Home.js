import { axios } from '@/utils/request'
const api = "/Index"
export default {
    infoStatistics:(parameter)=>{
        return axios({
            url: `${api}/infoStatistics`,
            method: 'post',
            data: parameter
        })
    },
    statistics:(parameter)=>{
        return axios({
            url: `${api}/statistics`,
            method: 'post',
            data: parameter
        })
    },
}