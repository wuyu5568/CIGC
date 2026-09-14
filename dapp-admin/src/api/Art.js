import { axios } from '@/utils/request'
const api = "/Article"
export default {
    getArticle:(parameter)=>{
        return axios({
            url: `${api}/getArticle`,
            method: 'post',
            data: parameter
        })
    },
    addArticle:(parameter)=>{
        return axios({
            url: `${api}/addArticle`,
            method: 'post',
            data: parameter
        })
    },
    deleteArticle:(parameter)=>{
        return axios({
            url: `${api}/deleteArticle`,
            method: 'post',
            data: parameter
        })
    },
    changeArticle:(parameter)=>{
        return axios({
            url: `${api}/changeArticle`,
            method: 'post',
            data: parameter
        })
    },
    getArticleDetails:(parameter)=>{
        return axios({
            url: `${api}/getArticleDetails`,
            method: 'post',
            data: parameter
        })
    },
}