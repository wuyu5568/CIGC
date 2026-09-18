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
  </van-nav-bar>
  <div class="page-main" v-if="item.id">
    <div class="hero" v-if="displayImage">
      <img :src="displayImage" alt="" />
    </div>
    <div class="title-block">
      <h1>{{ item.name || item.desc }}</h1>
      <p v-if="item.desc && item.desc !== item.name" class="short-desc">{{ item.desc }}</p>
    </div>
    <div class="sku-block" v-if="skus.length">
      <div class="sku-label">{{ lang('规格') }}</div>
      <div class="sku-list">
        <button
          v-for="sku in skus"
          :key="sku.id"
          type="button"
          class="sku-chip"
          :class="{ on: Number(selectedSkuId) === Number(sku.id) }"
          @click="selectedSkuId = Number(sku.id)"
        >
          <img v-if="sku.image" :src="sku.image" alt="" />
          <span class="sku-chip-text">
            <span>{{ sku.name }}</span>
            <em>{{ fmt(sku.amount) }} U</em>
          </span>
        </button>
      </div>
    </div>
    <ul class="investment-details">
      <li>
        <span class="detail-label">{{ lang('单价') }}</span>
        <span class="detail-value">{{ fmt(selectedAmount) }} USDT</span>
      </li>
      <li>
        <span class="detail-label">{{ lang('日封顶') }}</span>
        <span class="detail-value">{{ fmt(dailyCap) }} USDT<span v-if="capRange" class="cap-range">（{{ capRange }}）</span></span>
      </li>
    </ul>
      <div class="goods-detail">
        <span>{{ lang('商品详情') }}</span>
      </div>
      <div class="goods-html" v-if="item.detail" v-html="item.detail"></div>
      <p class="detail-empty" v-else>{{ lang('暂无详情') }}</p>
    <div class="buy-pad"></div>
  </div>
  <van-empty v-else-if="!loading" :description="lang('商品不存在')" />
  <div class="buy-bar" v-if="item.id">
    <div class="buy-price">{{ fmt(selectedAmount) }} <em>USDT</em></div>
    <button class="purchase-btn" :disabled="loading" @click="addToCart">{{ lang('加入购物车') }}</button>
  </div>
</div>
</template>
<script setup>
import { watch } from 'vue'
import lang from '@/i18n/index'
import request from "@/tools/request";
import { showFailToast, showSuccessToast } from "vant";
import { useRoute, useRouter } from 'vue-router'
import { displayAmount } from '@/tools/amount'
import { capForAmount, rangeForAmount, loadCapTiers, currentCapTiers } from '@/tools/dailyCap'
import { readCart, persistCart, enabledSKUs, changeCartQty } from '@/tools/web3Cart'

const router = useRouter()
const route = useRoute()
let loading = $ref(false)
let item = $ref({})
let selectedSkuId = $ref(0)

const fmt = (v) => displayAmount(v)
let capTiers = $ref(currentCapTiers())
const skus = $computed(() => enabledSKUs(item))
const selectedSku = $computed(() => skus.find((x) => Number(x.id) === Number(selectedSkuId)) || null)
const selectedAmount = $computed(() => (selectedSku && selectedSku.amount) || item.amount)
const displayImage = $computed(() => (selectedSku && selectedSku.image) || item.image || '')
const dailyCap = $computed(() => capForAmount(selectedAmount, capTiers) || item.daily_cap)
const capRange = $computed(() => rangeForAmount(selectedAmount, capTiers))

const handleBack = () => {
  router.push('/Web3Shop')
}

const addToCart = () => {
  if (!item?.id) {
    showFailToast(lang('商品信息错误'))
    return
  }
  if (skus.length && !selectedSku) {
    showFailToast(lang('请选择规格'))
    return
  }
  persistCart(changeCartQty(readCart(), item, 1, selectedSku))
  showSuccessToast(lang('已加入购物车'))
}

const fetchDetail = async () => {
  const id = Number(route.params.id)
  if (!id) {
    item = {}
    selectedSkuId = 0
    return
  }
  loading = true
  await request.get('admin/web3_goods_detail', {
    params: { id }
  }).then((res) => {
    if (Number(route.params.id) !== id) return
    if (res && res.status === 'ok' && res.item) {
      item = res.item
      const first = enabledSKUs(item)[0]
      selectedSkuId = first ? Number(first.id) : 0
    } else {
      item = {}
      showFailToast(res?.status || lang('商品不存在'))
    }
  }).catch(() => {
    if (Number(route.params.id) !== id) return
    item = {}
    showFailToast(lang('商品不存在'))
  }).finally(() => {
    loading = false
  })
}

watch(() => route.params.id, fetchDetail, { immediate: true })
loadCapTiers(request).then((rows) => {
  if (rows && rows.length) capTiers = rows
})
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
    .sku-block {
      margin: 0 0 16px;
      padding: 12px;
      background: rgba(34, 34, 34, 0.88);
      border: 1px solid #333;
      border-radius: 12px;
    }
    .sku-label {
      color: #fff;
      font-size: 14px;
      font-weight: 600;
      margin-bottom: 10px;
    }
    .sku-list {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }
    .sku-chip {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 10px 6px 6px;
      border: 1px solid #444;
      border-radius: 10px;
      background: #1f1f1f;
      color: #e8e8e8;
      font-size: 13px;
      text-align: left;
      img {
        width: 44px;
        height: 44px;
        border-radius: 6px;
        object-fit: cover;
        background: #2d2d2d;
        flex-shrink: 0;
      }
      .sku-chip-text {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        gap: 2px;
      }
      em {
        font-style: normal;
        color: #cab255;
        font-size: 12px;
      }
      &.on {
        border-color: #cab255;
        background: rgba(202, 178, 85, 0.12);
      }
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
        height: auto;
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
          .cap-range {
            color: #a0a0a0;
            font-size: 12px;
          }
        }
      }
    }
    .detail-section {
      margin-top: 4px;
      background: rgba(34, 34, 34, 0.88);
      border: 1px solid #333;
      border-radius: 12px;
      padding: 14px;
    }
    .goods-detail {
      display: flex;
      align-items: center;
      gap: 8px;
      margin: 0 0 12px;
      padding-bottom: 10px;
      color: #fff;
      font-size: 15px;
      font-weight: 600;
      line-height: 1;
      &::before {
        content: '';
        width: 3px;
        height: 14px;
        border-radius: 2px;
        background: #cab255;
        flex-shrink: 0;
      }
    }
    .detail-empty {
      margin: 0;
      padding: 18px 0 8px;
      color: #8a8a8a;
      font-size: 13px;
      text-align: center;
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
