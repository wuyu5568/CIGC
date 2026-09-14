<template>
  <div class="deal">
    <ul class="deal-menu" style="justify-content: center;">
      <li @click="tabMenu = 1" :class="{'selected-menu': tabMenu === 1}">IDO</li>
      <li @click="tabMenu = 2" :class="{'selected-menu': tabMenu === 2}">{{lang('认购记录')}}</li>
      <!-- <li @click="tabMenu = 3" :class="{'selected-menu': tabMenu === 3}">{{lang('收货地址')}}</li> -->
      <!-- <li @click="tabMenu = 3" :class="{'selected-menu': tabMenu === 3}">{{lang('充值记录')}}</li> -->
      <!-- <li @click="tabMenu = 4" :class="{'selected-menu': tabMenu === 4}">{{lang('提现')}}</li> -->
    </ul>
    <div class="deal-list" v-show="tabMenu === 1">
      <div class="product-list">
        <div class="product-item" v-for="item in userinfo.goods" @click="() => goShopDesc(item)">
          <div class="product-cover" :style="`box-sizing: border-box; background: url('${url}/images/${item.three}') no-repeat; background-size: cover;  background-position: center;`"></div>
          <div class="product-name">{{item.one}}</div>
          <!-- <div class="product-desc">{{ item.two }}</div> -->
          <div class="product-footer">
            <div class="product-amount">${{ item.four }}</div>
            <button class="shop-btn" :disabled="loading" @click.stop @click="() => usdtApproved ? transferUsdt(Number(item.four), Number(item.id)) : usdtApprove()">{{ loading ? lang('购买中...') : lang('购买') }}</button>
          </div>
        </div>
      </div>
    </div>
    <div class="deal-list" v-show="tabMenu === 2">
      <div class="order-list" v-for="(item, index) in orderList" :key="index">
        <div class="order-box">
          <div class="order-column">
            <p>{{lang('金额')}}</p>
            <p>{{ item.amount }}</p>
          </div>
          <div class="order-column">
            <p>{{lang('状态')}}</p>
            <p>{{ item.status === '1' ? '收益中' : item.status === '2' ? '已出局' : '暂无' }}</p>
          </div>
          <div class="order-column">
            <p>{{lang('时间')}}</p>
            <p>{{ item.createdAt }}</p>
          </div>
        </div>
        <div class="order-box">
          <div class="order-column">
            <p>{{lang('已释放')}}</p>
            <p>{{ item.amountGet }}</p>
          </div>
          <div class="order-column">
            <p>{{lang('待释放')}}</p>
            <p>{{ item.amountLast }}</p>
          </div>
          <div class="order-column">
            <p>{{lang('倍率')}}</p>
            <p>{{ item.status }}</p>
          </div>
        </div>
        <div class="order-box-2">
          <div class="order-column">
            <p><span>{{ lang('名称') }}: </span>{{ item.four }}</p>
          </div>
          <div class="order-column">
            <p><span>{{ lang('收货人') }}: </span>{{ item.three }}</p>
          </div>
          <div class="order-column">
            <p><span>{{ lang('收货地址') }}: </span>{{ item.one }}</p>
          </div>
        </div>
      </div>
      <Pagination
        v-model="allPage"
        :page-count="allPageCount"
        mode="simple"
        @change="getOrderList"
      />
    </div>
    <Exchange v-if="isPopupOpened" @close="changePopup(false)" />
  </div>
</template>

<script setup lang="ts">
import Exchange from './exchange.vue'
import userPerson from "@/pinia/person";
import userSystem from "@/pinia/system";
import { Contract, ETH } from "@/tools/contract";
import lang from '@/i18n/index'
import BiwMeta from '@/services/index'
import request from "@/tools/request";
import { showLoadingToast, closeToast, showFailToast, showDialog, closeDialog, showSuccessToast } from "vant";
import { useRouter } from 'vue-router'
import { ref, computed, onMounted } from 'vue'
const USDT: Contract = new Contract(import.meta.env.VITE_USDT, "ERC20");
const BUY: Contract = new Contract(import.meta.env.VITE_BUY, "BUY");

const router = useRouter()

interface ProductItem {
  id: string,
  one: string,
  two: string,
  three: string,
  four: string
}

interface UserInfo {
  one: string,
  two: string,
  three: string,
  four: string,
  five: string,
  six: string,
  goods: ProductItem[];
  buyNumSix: number;
  sellNumSix: number;
  buyNumOne: number;
  sellNumOne: number;
  buyNumTwo: number;
  sellNumTwo: number;
  buyNumFour: number;
  sellNumFour: number;
  buyNumFive: number;
  sellNumFive: number;
  // 其他字段...
}

let loading = ref(false);
const system = userSystem();
const person = userPerson();
const url = import.meta.env.VITE_API
const userinfo = computed<UserInfo>(() => person.userinfo);
const biwSign = computed(() => person.sign);
let usdtBalance = ref("0");
let usdtApproved = ref(false);
let orderList: any[] = ref([]);
let allPage = ref(1);
let allPageCount = ref(1);

const props = defineProps({
  menu: Number,
});

const incomeType: { [key: string]: string } = {
  '1': lang('挖矿收益'),
  '2': lang('分享收益'),
  '3': lang('前四名收益'),
  '4': lang('矩阵收益'),
  '5': lang('兑换'),
  '6': lang('提现'),
  '7': lang('认购')
}

const getOrderList = async (page: number = 1) => {
  await request.get("app_server/order_list", {
      params: {
        page
      }
    }).then((res: any) => {
      allPageCount.value = Math.ceil(res.count / 10);
      orderList.value = res.list
    })
}

const goShopDesc = (item: ProductItem) => {
  localStorage.setItem('goods', JSON.stringify(item))
  router.push('/shop')
}

onMounted(() => {
  getOrderList()
  getUsdtApproved()
})

/* 获取授权 */
const getUsdtApproved = async () => {
    closeToast()
    let res = await USDT.call("allowance", [ETH.account, BUY.address]);
    console.log('getUsdtApproved', Number(res))
    usdtApproved.value = Number(res) > 0;

    return usdtApproved.value
}

const usdtApprove = async () => {
    showLoadingToast({
        message: lang('授权中'), duration: 0, overlay: true, overlayStyle: {
            background: "transparent"
        }
    });
    await USDT.send("approve", [
        BUY.address,
        "115792089237316195423570985008687907853269984665640564039457584007913129639935"
    ]).then(getUsdtApproved).catch(() => closeToast());
}

/* 获取代币余额 */
const getBalance = async () => {
    const res = await ETH.getUSDTBalance()
    usdtBalance.value = String(res);
}

const getData = async () => {
    showLoadingToast({
        message: lang('授权中'), duration: 0, overlay: true, overlayStyle: {
            background: "transparent"
        }
    });
    await getBalance();
    await getUsdtApproved();
    closeToast();
}

// getData()

/* USDT转账 */
const transferUsdt = async (count: number, id: number) => {
  loading.value = true;
  person.userinfo.status = 'running'
  BUY.send("buy", [count, id],).then(() => {
      loading.value = false;
      system.setBuyTime();
      showDialog({
        title: lang('提示'),
        message: lang(`USDT 转账成功！`),
        theme: 'round-button',
        confirmButtonColor: "#242738",
        confirmButtonText: lang('我知道了！'),
      })
  }).catch(() => loading.value = false);
}

let tabMenu = ref(props.menu === 8 ? 2 : 1)
let isPopupOpened = ref(false)

const changePopup = (value: boolean) => {
  isPopupOpened.value = value
}
</script>
<style scoped lang="less">
@import "../styles/deal.less";

/* 按钮样式 */
.shop-btn {
  padding: 5px 15px;
  background-color: #F9D02A;
  color: #000;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  font-weight: bold;
}

.shop-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
