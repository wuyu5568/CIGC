import { createRouter, createWebHashHistory, Component } from "vue-router";
import Index from "@/views/index/index.vue";
import IdoDetails from '@/views/ido/details.vue'
import Profile from '@/views/profile/index.vue'
import Payment from '@/views/payment/index.vue'
import Count from '@/views/share/count.vue'
import Withdrawal from '@/views/withdrawal/index.vue'
import Home from '@/views/home/index.vue'
// import Recharge from '@/views/recharge/index.vue'
import Trade from '@/views/trade/index.vue'
import Node from '@/views/node/index.vue'
import Contact from '@/views/contact/index.vue'
import Address from '@/views/index/address.vue'
import Shop from '@/views/subpage/shop.vue'
import Order from '@/views/subpage/order.vue'
import PowerShop from '@/views/subpage/powerShop.vue'
import Web3Shop from '@/views/subpage/web3Shop.vue'
import Stat from '@/views/subpage/stat.vue'
import Wallet from "@/views/subpage/wallet.vue";
import Community from '@/views/subpage/community.vue'
import Level from '@/views/subpage/level.vue'
import Withdraw from '@/views/subpage/withdraw.vue'
import Recharge from '@/views/subpage/recharge.vue'
import Transfer from '@/views/subpage/transfer.vue'
import PowerOrder from '@/views/subpage/powerOrder.vue'

// 定义路由
const routes = [
    { path: '/', component: Index },
    { path: '/home', component: Home },
    { path: '/recharge', component: Recharge},
    { path: '/idoDetails', component: IdoDetails },
    { path: '/profile', component: Profile },
    { path: '/payment', component: Payment },
    { path: '/count', component: Count},
    { path: '/withdrawal', component: Withdrawal},
    { path: '/withdraw/:type', component: Withdraw},
    { path: '/trade', component: Trade},
    { path: '/transfer', component: Transfer},
    { path: '/node', component: Node},
    { path: '/contact', component: Contact},
    { path: '/address', component: Address},
    { path: '/shop', component: Shop},
    { path: '/Web3Shop', component: Web3Shop},
    { path: '/order', component: Order},
    { path: '/order/:id', component: Order},
    { path: '/powerShop', component: PowerShop},
    { path: '/stat', component: Stat},
    { path: '/wallet', component: Wallet},
    { path: '/community', component: Community},
    { path: '/level', component: Level},
    { path: '/powerOrder', component: PowerOrder}
];

// 创建路由实例
const router = createRouter({
  history: createWebHashHistory(),
  routes
});

export default router;