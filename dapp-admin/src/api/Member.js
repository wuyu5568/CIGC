import { axios } from '@/utils/request'
const api = "/User"
export default {
    index:(parameter)=>{
        return axios({
            url: `${api}/index`,
            method: 'post',
            data: parameter
        })
    },
    getUserDetails:(parameter)=>{
        return axios({
            url: `${api}/getUserDetails`,
            method: 'post',
            data: parameter
        })
    },
    updateUser:(parameter)=>{
        return axios({
            url: `${api}/updateUser`,
            method: 'post',
            data: parameter
        })
    },
    nodeList:(parameter)=>{
        return axios({
            url: `${api}/nodeList`,
            method: 'post',
            data: parameter
        })
    },
    superNodeList:(parameter)=>{
        return axios({
            url: `${api}/superNodeList`,
            method: 'post',
            data: parameter
        })
    },
    runUpdateUserMoney:(parameter)=>{
        return axios({
            url: `${api}/runUpdateUserMoney`,
            method: 'post',
            data: parameter
        })
    },
    getDownUserList:(parameter)=>{
        return axios({
            url: `${api}/getDownUserList`,
            method: 'post',
            data: parameter
        })
    },
    addNodeLog:(parameter)=>{
        return axios({
            url: `${api}/addNodeLog`,
            method: 'post',
            data: parameter
        })
    },
}