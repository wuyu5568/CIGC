<template>
  <div class="my">
    <div class="content-box">
      <div class="income-box">
        <div class="income-main">
          <p>{{lang('待领取收益')}}</p>
          <p>${{userinfo.amountGetSub || 0}}</p>
        </div>
        <div class="income-footer">
          <!-- <div class="income-footer-item">
            <p>0</p>
            <p>KEY</p>
          </div> -->
          <div class="income-footer-item">
            <p>0</p>
            <p>ISPAY</p>
          </div>
          <div class="income-footer-item">
            <p>0</p>
            <p>USDT</p>
          </div>
        </div>
      </div>
    </div>
    <!-- <div class="content-box">
      <div class="invite-box">
        <div class="income-box-item" @click="router.push('/count')">
          <span class="count">{{lang('邀请数量')}}</span>
          <i class="right-arrow-icon"></i>
        </div>
        <div class="income-box-item" @click="router.push('/withdrawal')">
          <span class="withdrawal">{{lang('提现')}}</span>
          <i class="right-arrow-icon"></i>
        </div>
      </div>
    </div> -->
    <!-- <button class="recharge-button" @click="openShopUrl">{{ lang("充值") }}</button> -->
    <div class="withdrawal-btn" @click="router.push('/withdrawal')">
      <span class="withdrawal">{{lang('提现')}}</span>
    </div>
    <div class="content-box">
      <div class="asset-box">
        <!-- <div class="asset-header">
          <p>{{lang('总资产')}}</p>
          <p>${{ totalAsset }}</p>
        </div> -->
        <ul class="asset-tab-menu">
          <li :class="{'active-tab': tabMenu === 1}" @click="tabMenu = 1">{{lang('充值资产')}}</li>
          <li :class="{'active-tab': tabMenu === 2}" @click="tabMenu = 2">{{lang('合约资产')}}</li>
        </ul>
        <!-- <div class="asset-item">
          <span>{{tabMenu === 1 ? lang('充值资产') : lang('合约资产')}}</span>
          <span>${{formatNumber(tabMenu === 1 ? ispsUsdt : contractAsset)}}</span>
        </div> -->
        <div class="asset-list">
          <div class="asset-list-item">
            <div class="asset-list-item-left">
              <div class="asset-list-item-cover ustd-cover"></div>
              <span>USDT</span>
            </div>
            <div class="asset-list-item-amount">
              <span>{{ formatNumber(userinfo.usdt) }}</span>
              <!-- <span>0 USDT</span> -->
            </div>
          </div>
          <div class="asset-list-item">
            <div class="asset-list-item-left">
              <div class="asset-list-item-cover ispay-cover"></div>
              <span>ISPAY</span>
            </div>
            <div class="asset-list-item-amount">
              <span>{{ userinfo.raw.replace(/[^\d.]/g, '') || 0 }}</span>
              <!-- <span>0 USDT</span> -->
            </div>
          </div>
        </div>
      </div>
    </div>
    <!-- <div class="address-modal" @click="showShopUrl = false" v-if="showShopUrl">
      <div class="address-modal-main" @click.stop>
        <i class="modal-close-btn" @click="showShopUrl = false"></i>
        <p class="address-title">{{lang('购买地址')}}</p>
        <p class="address-text">USDT{{lang('地址')}}：{{ userinfo.address }}</p>
        <button class="address-copy-btn" @click="copyToClipboard(userinfo.address)">{{lang('复制地址')}}</button>
        <p class="address-text">BIW{{lang('地址')}}：{{ userinfo.addressBiw }}</p>
        <button class="address-copy-btn" @click="copyToClipboard(userinfo.addressBiw)">{{lang('复制地址')}}</button>
      </div>
    </div> -->
  </div>
</template>
<script setup lang="ts">
import { ETH } from '@/tools/contract'
import userPerson from "@/pinia/person";
import lang from '@/i18n/index'
import copy from 'copy-to-clipboard';
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { ref, computed } from 'vue'

const router = useRouter()
const person = userPerson();
const userinfo = computed(() => person.userinfo);

const contractAsset = computed(() => {
  return (Number(formatNumber(userinfo.value.balanceBiw)) * Number(userinfo.value.biwPrice) + Number(userinfo.value.balanceUsdt)).toFixed(2)
})

const totalAsset = computed(() => {
  return (Number(ispsUsdt.value) + (Number(formatNumber(userinfo.value.balanceBiw)) * Number(userinfo.value.biwPrice) + Number(userinfo.value.balanceUsdt))).toFixed(2)
})

const copyToClipboard = (text: string) => {
  copy(text);

  showToast({
    message: lang("地址已复制到剪贴板"),
    position: 'center',
    duration: 2000,
  });
}

let showShopUrl = ref(false)
let tabMenu = ref(1)
let usdt: string = ref('0.0')
let isps: string = ref('0.0')
let ispsUsdt = ref('0.0')

const openShopUrl = () => {
  console.log(44444)
  showShopUrl.value = true
}
const formatNumber = (number: number | string) => {
  // 将参数转换为数字类型
  const num = parseFloat(String(number));
  // 如果无法转换为有效数字，则返回原始参数
  if (isNaN(num)) {
      return number;
  }
  // 将数字保留两位小数
  return num.toFixed(2);
}

const getAccount = async () => {
  const usdtBalance = await ETH.getUSDTBalance()
  // console.log('usdtBalance', usdtBalance, 'ispsBalance', ispsBalance, 'ispsCost', ispsCost)
  usdt.value = String(usdtBalance)
}

getAccount()

const tabMenuChange = (index: number) => {
  tabMenu.value = index
}
</script>
<style scoped lang="less">
@import "../styles/my.less";
</style>
