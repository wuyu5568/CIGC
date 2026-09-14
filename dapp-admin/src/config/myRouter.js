import { BasicLayout } from '@/layouts'
export const asyncRouterMap = [
    {
        path: '/',
        name: 'index',
        component: BasicLayout,
        meta: { title: '首页' },
        redirect: '/home',
        children: [
            /*首页*/
            {
                path: '/home',
                name: 'home',
                component: () => import('@/views/home/index'),
                meta: { title: '数据统计', keepAlive: true, icon: "pie-chart", permission: ['dashboard'] },
            },
            {
                path: '/ordersList',
                name: 'ordersList',
                component: () => import('@/views/orders/ordersList'),
                meta: { title: '分红数据', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            },
            {
                path: '/recharge',
                name: 'recharge',
                component: () => import('@/views/orders/recharge'),
                meta: { title: '充值列表', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            },
            {
                path: '/member',
                name: 'member',
                component: () => import('@/views/orders/member'),
                meta: { title: '用户数据', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            },
            {
                path: '/dealList',
                name: 'dealList',
                component: () => import('@/views/orders/dealList'),
                meta: { title: '商品列表', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            },
            {
                path: '/dealList2',
                name: 'dealList2',
                component: () => import('@/views/orders/dealList2'),
                meta: { title: '医疗保健商品', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            },
            {
                path: '/dealList3',
                name: 'dealList3',
                component: () => import('@/views/orders/dealList3'),
                meta: { title: 'Web3商品列表', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            },
            {
                path: '/subscription',
                name: 'subscription',
                component: () => import('@/views/orders/subscription'),
                meta: { title: '认购列表', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            },
            {
                path: '/subscription4',
                name: 'subscription4',
                component: () => import('@/views/orders/subscription4'),
                meta: { title: '长寿商店认购列表', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            },
            {
                path: '/subscription2',
                name: 'subscription2',
                component: () => import('@/views/orders/subscription2'),
                meta: { title: '医疗保健商品认购列表', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            },
            {
                path: '/subscription3',
                name: 'subscription3',
                component: () => import('@/views/orders/subscription3'),
                meta: { title: 'Web3商品认购列表', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            },
            // {
            //     path: '/location',
            //     name: 'location',
            //     component: () => import('@/views/orders/location'),
            //     meta: { title: '出局列表', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            // },
            // {
            //     path: '/investor',
            //     name: 'investor',
            //     component: () => import('@/views/orders/investor'),
            //     meta: { title: '投资列表', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            // },
            // {
            //     path: '/location2',
            //     name: 'location2',
            //     component: () => import('@/views/orders/location2'),
            //     meta: { title: '会员投资列表', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            // },
            {
                path: '/withdraw',
                name: 'withdraw',
                component: () => import('@/views/orders/withdraw'),
                meta: { title: '提现列表', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            },
            {
                path: '/lookChildren',
                name: 'lookChildren',
                component: () => import('@/views/member/lookChildren'),
                hidden:true,
                meta: { title: '查看下级', keepAlive: true, permission: ['dashboard'] },
            },
            // {
            //     path: '/month_recommend',
            //     name: 'month_recommend',
            //     component: () => import('@/views/orders/month_recommend'),
            //     meta: { title: '始勇者', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            // },
            // {
            //     path: '/dealList',
            //     name: 'dealList',
            //     component: () => import('@/views/orders/dealList'),
            //     meta: { title: '交易列表', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            // },
            // {
            //     path: '/configCSD',
            //     name: 'configCSD',
            //     component: () => import('@/views/orders/configCSD'),
            //     meta: { title: 'csd价格设置', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            // },
            {
                path: '/config',
                name: 'config',
                component: () => import('@/views/orders/config'),
                meta: { title: '配置项', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            },
            // {
            //     path: '/admin',
            //     name: 'admin',
            //     component: () => import('@/views/orders/admin'),
            //     meta: { title: '管理员管理', keepAlive: true, icon: "setting", permission: ['dashboard'] },
            // },
        ]
    },
    {
        path: '*', redirect: '/404', hidden: true
    }
]