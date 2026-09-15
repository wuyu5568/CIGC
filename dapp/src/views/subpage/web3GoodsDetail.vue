<template>
<div class='shop-page'>
  <van-nav-bar
    :title="item.name || lang('商品详情')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
    @click-right="router.push('order/1')"
  >
    <template #right>
      <span class="nav-order">
        <van-icon name="orders-o" />
        {{ lang('商城订单') }}
      </span>
    </template>
  </van-nav-bar>
  <div class="page-main" v-if="item.id">
    <div class="hero" v-if="item.image">
      <img :src="item.image" alt="" />
    </div>
    <div class="title-block">
      <h1>{{ item.name || item.desc }}</h1>
      <p v-if="item.desc && item.desc !== item.name" class="short-desc">{{ item.desc }}</p>
    </div>
    <ul class="investment-details">
      <li>
        <span class="detail-label">{{ lang('单价') }}</span>
        <span class="detail-value">{{ fmt(item.amount) }} USDT</span>
      </li>
      <li>
        <span class="detail-label">{{ lang('日封顶') }}</span>
        <span class="detail-value">{{ fmt(item.daily_cap) }} USDT</span>
      </li>
    </ul>
    <div class="goods-html" v-if="item.detail" v-html="item.detail"></div>
    <div class="buy-pad"></div>
  </div>
  <van-empty v-else-if="!loading" :description="lang('商品不存在')" />
  <div class="buy-bar" v-if="item.id">
    <div class="buy-price">{{ fmt(item.amount) }} <em>USDT</em></div>
    <button class="purchase-btn" :disabled="loading" @click="isOpen = true">{{ lang('购买') }}</button>
  </div>
  <Web3BuyModal
    v-model="isOpen"
    :goods-id="item.id"
    :amount="item.amount"
    :spot="spot"
  />
</div>
</template>
<script setup>
import { watch } from 'vue'
import userPerson from "@/pinia/person";
import lang from '@/i18n/index'
import request from "@/tools/request";
import { showFailToast } from "vant";
import { useRoute, useRouter } from 'vue-router'
import { displayAmount } from '@/tools/amount'
import Web3BuyModal from './components/web3BuyModal.vue'

const router = useRouter()
const route = useRoute()
const person = userPerson();
const userinfo = $computed(() => person.userinfo);
let loading = $ref(false)
let item = $ref({})
const price = $ref('')
const isOpen = $ref(false)
const spot = $computed(() => price || userinfo.ispayPrice || '2000')

const fmt = (v) => displayAmount(v)

const handleBack = () => {
  router.push('/Web3Shop')
}

const fetchPrice = () => {
  request.get('app_server/ispay_price').then((res) => {
    if (res && res.price) price = res.price
  }).catch(() => {})
}

const fetchDetail = async () => {
  const id = Number(route.params.id)
  if (!id) {
    item = {}
    showFailToast(lang('商品不存在'))
    return
  }
  loading = true
  await request.get('admin/web3_goods_detail', {
    params: { id }
  }).then((res) => {
    if (res && res.status === 'ok' && res.item) {
      item = res.item
    } else {
      item = {}
      showFailToast(res?.status || lang('商品不存在'))
    }
  }).catch(() => {
    item = {}
    showFailToast(lang('商品不存在'))
  }).finally(() => {
    loading = false
  })
}

watch(() => route.params.id, fetchDetail, { immediate: true })
fetchPrice()
</script>
<style lang='less' scoped>
  .shop-page {
    min-height: 100vh;
    background: #121212;
    .page-main {
      width: 100%;
      padding: 60px 15px 90px;
      box-sizing: border-box;
    }
    .hero {
      width: 100%;
      border-radius: 12px;
      overflow: hidden;
      background: #2d2d2d;
      margin-bottom: 16px;
      img {
        display: block;
        width: 100%;
        max-height: 280px;
        object-fit: cover;
      }
    }
    .title-block {
      h1 {
        margin: 0 0 8px;
        font-size: 20px;
        color: #fff;
        font-weight: 600;
      }
      .short-desc {
        margin: 0 0 12px;
        color: #a0a0a0;
        font-size: 13px;
        line-height: 1.5;
      }
    }
    .investment-details {
      list-style: none;
      margin: 0 0 16px;
      padding: 0 4px;
      background: rgba(34, 34, 34, 0.88);
      border: 1px solid #333;
      border-radius: 12px;
      li {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 12px;
        border-bottom: 1px solid #2d2d2d;
        &:last-child {
          border-bottom: none;
        }
        .detail-label {
          color: #a0a0a0;
          font-size: 14px;
        }
        .detail-value {
          color: #cab255;
          font-size: 14px;
        }
      }
    }
    .goods-html {
      color: #e0e0e0;
      font-size: 14px;
      line-height: 1.7;
      word-break: break-word;
      :deep(img) {
        max-width: 100%;
        height: auto;
        display: block;
        margin: 8px 0;
        border-radius: 8px;
      }
      :deep(p) {
        margin: 0 0 10px;
      }
      :deep(table) {
        width: 100%;
        border-collapse: collapse;
      }
      :deep(a) {
        color: #cab255;
      }
    }
    .buy-pad {
      height: 20px;
    }
    .buy-bar {
      position: fixed;
      left: 0;
      right: 0;
      bottom: 0;
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 10px 15px calc(10px + env(safe-area-inset-bottom));
      background: rgba(18, 18, 18, 0.96);
      border-top: 1px solid #333;
      .buy-price {
        flex: 1;
        color: #cab255;
        font-size: 18px;
        font-weight: 600;
        em {
          font-style: normal;
          font-size: 12px;
          color: #9a9a9a;
          font-weight: 400;
        }
      }
      .purchase-btn {
        min-width: 120px;
        padding: 12px 20px;
        background-color: #cab255;
        color: #121212;
        border: none;
        border-radius: 8px;
        font-size: 16px;
        font-weight: 500;
      }
    }
    .nav-order {
      display: inline-flex;
      align-items: center;
      gap: 4px;
      font-size: 14px;
      color: rgb(204, 204, 204);
      .van-icon {
        font-size: 16px;
      }
    }
  }
</style>
