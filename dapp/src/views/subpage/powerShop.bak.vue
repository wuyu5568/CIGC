<template>
<div class='shop-page'>
  <van-nav-bar
    :title="lang('长寿商城')"
    :right-text="lang('收货地址')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
    @click-right="changeShippingAddress(true)"
  />
  <div class="page-main">
    <div class="page-title">
      <h2>{{ lang('长寿商城') }}</h2>
      <p>{{ lang('通过购买商品获得对应的原发算力') }}</p>
    </div>
    <div class="shop-list">
      <div class="shop-item" v-for="item in userinfo.goodsTwo">
        <div class="shop-cover" :style="`box-sizing: border-box; background: url('/images/${item.three}') no-repeat; background-size: cover;  background-position: center;`"></div>
        <div class="shop-name">{{item.one}}</div>
        <div class="shop-price">${{ item.four }}</div>
        <button class="shop-btn" :disabled="loading" @click.stop @click="handleShop(item)">{{ lang('购买') }}</button>
      </div>
    </div>
  </div>
  <van-popup
    v-model:show="showShippingAddress"
    position="right"
    :duration="0.2"
    :style="{ width: '100%', height: '100%', background: '#171C21' }"
  >
    <ShippingAddressDialog :changeShippingAddress="changeShippingAddress" />
  </van-popup>
  <van-action-bar>
    <van-action-bar-button type="danger" :text="lang('商城订单')" @click="router.push('order/2')" />
  </van-action-bar>
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import lang from '@/i18n/index'
import request from "@/tools/request";
import { showLoadingToast, showConfirmDialog, closeToast, showFailToast, showDialog, closeDialog, showSuccessToast } from "vant";
import { useRouter } from 'vue-router'
import ShippingAddressDialog from "./components/shippingAddressDialog.vue";

const actions = [
  {
    name: '100% USDT',
    value: '1'
  },
  {
    name: '80% USDT，20% BRC20',
    value: '0'
  }
]

const router = useRouter()

let loading = $ref(false);
const person = userPerson();
const userinfo = $computed(() => person.userinfo);
const sign = $computed(() => person.sign);
const id = $ref(null)
const show = $ref(false)
const showShippingAddress = $ref(false)

const handleShop = async (item) => {
  if (userinfo.userAddress && userinfo.userAddress.length === 0) {
    showDialog({
      message: lang('未添加收货地址，无法购买'),
      theme: 'round-button',
      confirmButtonText: lang('添加收货地址')
    }).then(() => {
      changeShippingAddress(true)
    });
    return
  }

  showConfirmDialog({
    title: null,
    confirmButtonText: lang('确定'),
    cancelButtonText: lang('取消'),
    message: `${lang('是否确认购买')}\n\nUSDT：${ userinfo.amountUsdt || 0.0000 }    BRC20：${ userinfo.rawNew || 0.0000 }`,
  }).then(async () => {
    showLoadingToast()
    await request.post("app_server/buy_two", {
      id: item.id,
      addressId: userinfo.userAddress[0].id,
      sign: sign
    }).then((res) => {
      if (res.status === 'ok') {
        showSuccessToast(lang('购买成功'))
      } else if (res.status === '用户已锁定') {
        showFailToast(lang('用户已锁定'))
      } else {
        showFailToast(lang('购买失败'))
      }
      console.log(res)
    }).catch((error) => {
      showFailToast(lang('购买失败'))
    })
  })
}

const changeShippingAddress = (value) => {
  showShippingAddress = value
}


const handleBack = () => {
  router.back()
}

</script>
<style lang='less' scoped>
  .shop-page {
    min-height: 100vh;
    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 0;
      width: 100%;
      height: 200px;
      background: url('../../assets/images/shop-banner.jpg') no-repeat center bottom;
      background-size: cover;
      z-index: -1;
    }
    .page-main {
      width: 100%;
      padding: 40px 15px 80px 15px;
      box-sizing: border-box;
      .page-title {
        padding: 30px 0;
        color: #FFF;
        h2 {
          font-size: 16px;
        }
        p {
          width: 130px;
          line-height: 1.6;
        }
      }
      .shop-list {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
        .shop-item {
          width: calc(50% - 5px);
          box-sizing: border-box;
          background: #171c21;
          border: 1px solid #33383f;
          border-radius: 8px;
          padding: 10px;
          display: flex;
          flex-direction: column;
          gap: 10px;
          .shop-cover {
            height: 140px;
            display: flex;
            align-items: center;
            justify-content: center;
          }
          .shop-price {
            font-size: 14px;
          }
          .shop-btn {
            background: rgba(255, 255, 255, .2);
            border-radius: 6px;
            border: 0;
            height: 30px;
            text-align: center;
            color: #ffd127;
          }
        }
      }
    }
    /deep/ .van-action-bar {
      padding: 0 30px;
    }
    /deep/ .van-button {
      border-radius: 12px;
    }
  }
</style>