import { axios } from '@/utils/request'
const api = "/Auth"
export default {
    getAllAdmin:(parameter)=>{
        return axios({
            url: `${api}/getAllAdmin`,
            method: 'post',
            data: parameter
        })
    },
    changeInfo:(parameter)=>{
        return axios({
            url: `${api}/changeInfo`,
            method: 'post',
            data: parameter
        })
    },
    changeJuese:(parameter)=>{
        return axios({
            url: `${api}/changeJuese`,
            method: 'post',
            data: parameter
        })
    },
    addAdminMember:(parameter)=>{
        return axios({
            url: `${api}/addAdminMember`,
            method: 'post',
            data: parameter
        })
    },
    getGroup:(parameter)=>{
        return axios({
            url: `${api}/getGroup`,
            method: 'post',
            data: parameter
        })
    },
    deleteGroup:(parameter)=>{
        return axios({
            url: `${api}/deleteGroup`,
            method: 'post',
            data: parameter
        })
    },
    changeGroup:(parameter)=>{
        return axios({
            url: `${api}/changeGroup`,
            method: 'post',
            data: parameter
        })
    },
    getAllRules:(parameter)=>{
        return axios({
            url: `${api}/getAllRules`,
            method: 'post',
            data: parameter
        })
    },
    addGroup:(parameter)=>{
        return axios({
            url: `${api}/addGroup`,
            method: 'post',
            data: parameter
        })
    },
    getGroupDetail:(parameter)=>{
        return axios({
            url: `${api}/getGroupDetail`,
            method: 'post',
            data: parameter
        })
    },
}