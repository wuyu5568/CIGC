import { axios } from '@/utils/request'
const api = "/Project"
export default {
    getProjectType: (parameter) => {
        return axios({
            url: `${api}/getProjectType`,
            method: 'post',
            data: parameter
        }) 
    },
    updateProjectType: (parameter) => {
        return axios({
            url: `${api}/updateProjectType`,
            method: 'post',
            data: parameter
        })
    },
    getAllProject: (parameter) => {
        return axios({
            url: `${api}/getAllProject`,
            method: 'post',
            data: parameter
        })
    },
    updatePic: (parameter) => {
        return axios({
            url: `${api}/updatePic`,
            method: 'post',
            data: parameter
        })
    },
    addProject: (parameter) => {
        return axios({
            url: `${api}/addProject`,
            method: 'post',
            data: parameter
        })
    },
    getSaleList: (parameter) => {
        return axios({
            url: `${api}/getSaleList`,
            method: 'post',
            data: parameter
        })
    },
    addSaleOrder: (parameter) => {
        return axios({
            url: `${api}/addSaleOrder`,
            method: 'post',
            data: parameter
        })
    }, 
    getReserveList: (parameter) => {
        return axios({
            url: `${api}/getReserveList`,
            method: 'post',
            data: parameter
        })
    },
    getProjectLog: (parameter) => {
        return axios({
            url: `${api}/getProjectLog`,
            method: 'post',
            data: parameter
        })
    },
    getSaleList: (parameter) => {
        return axios({
            url: `${api}/getSaleList`,
            method: 'post',
            data: parameter
        })
    },
    updateSaleStatus: (parameter) => {
        return axios({
            url: `${api}/updateSaleStatus`,
            method: 'post',
            data: parameter
        })
    },
    deleteSaleLog: (parameter) => {
        return axios({
            url: `${api}/deleteSaleLog`,
            method: 'post',
            data: parameter
        })
    },
}