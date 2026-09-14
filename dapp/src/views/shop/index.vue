<template>
<div class="shop-page-container">
  <ChildrenHeader :isHideSet="true" />
  <div class='shop-page'>
    <div class="shop-cover"><img style="width: 100%" :src="`${url}/static/${product.three}`" /></div>
    <div class="shop-title">{{ product.one }}</div>
    <!-- <div class="shop-desc">{{ product.two }}</div> -->
    <div class="shop-price">${{ product.four }}</div>
    <div class="shop-intro">{{ product.five }}</div>
    <button class="shop-btn" :disabled="loading" @click="() => usdtApproved ? transferUsdt(Number(product.four)) : usdtApprove()">{{ loading ? lang('购买中...') : lang('购买') }}</button>
  </div>
</div>
</template>
<script setup lang="ts">
import { Contract, ETH } from "@/tools/contract";
import { showLoadingToast, closeToast, showFailToast, showDialog, closeDialog, showSuccessToast } from "vant";
import userPerson from "@/pinia/person";
import userSystem from "@/pinia/system";
const USDT: Contract = new Contract(import.meta.env.VITE_USDT, "ERC20");
const BUY: Contract = new Contract(import.meta.env.VITE_BUY, "BUY");
import ChildrenHeader from '../../components/header/childrenHeader2.vue'
import lang from '@/i18n/index'
import { ref, computed, onMounted } from 'vue'

const url = import.meta.env.VITE_API

let loading = ref(false);
const system = userSystem();
const person = userPerson();
const userinfo = computed(() => person.userinfo);
const biwSign = computed(() => person.sign);
let usdtApproved = ref(false);

const product = JSON.parse(localStorage.getItem('goods') || '{}')
/* 获取授权 */
const getUsdtApproved = async () => {
    closeToast()
    let res = await USDT.call("allowance", [ETH.account, BUY.address]);
    console.log('getUsdtApproved', Number(res))
    usdtApproved.value = Number(res) > 0;

    return usdtApproved.value
}

onMounted(() => {
  getUsdtApproved()
})

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


const transferUsdt = async (count: number) => {
  if (userinfo.value.one) {
    return showFailToast(lang('请先填写收货地址'));
  }
  loading.value = true;
  person.userinfo.status = 'running'
  BUY.send("buy", [count],).then(() => {
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
</script>
<style lang='less' scoped>
.shop-page-container {
  width: 100%;
  min-height: 100vh;
}

.shop-page {
  .shop-cover {
    aspect-ratio: 16 / 11;
    margin-bottom: 10px;
  }
  .shop-title {
    font-size: 20px;
    line-height: 40px;
    padding: 0 20px;
  }
  .shop-desc {
    font-size: 16px;
    line-height: 32px;
    padding: 0 20px;
  }
  .shop-price {
    font-size: 20px;
    line-height: 32px;
    padding: 0 20px;
    font-weight: 600;
  }
  .shop-intro {
    font-size: 16px;
    line-height: 32px;
    padding: 0 20px;
    margin-top: 20px;
  }
  .shop-btn {
    width: 90%;
    height: 40px;
    line-height: 40px;
    background: #F8B204;
    border: 0;
    border-radius: 4px;
    display: inline-block;
    font-size: 16px;
    opacity: .9;
    font-weight: 500;
    margin-left: 20px;
    margin-top: 30px;
    cursor: pointer;
  }
  .shop-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}
</style>