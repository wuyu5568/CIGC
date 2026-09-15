<template>
<div class='shop-page'>
  <van-nav-bar
    :title="lang('Web3商城')"
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
  <div class="page-main">
    <div class="page-title">
      <span class="price-label">{{ lang('现价') }}</span>
      <span class="price-value">{{ spot }}<em>U / ispay</em></span>
    </div>
    <div class="shop-list">
      <div class="investment-card" v-for="item in list" :key="item.id" @click="openDetail(item)">
        <div class="card-head">
          <div class="card-cover" v-if="item.image">
            <img :src="item.image" alt="" />
          </div>
          <div class="card-head-text">
            <div class="card-main-title">{{ item.name || item.desc }}</div>
            <div class="card-desc" v-if="item.desc && item.desc !== item.name">{{ item.desc }}</div>
          </div>
        </div>
        <ul class="investment-details">
          <li>
            <span class="detail-label">{{ lang('单价') }}：</span>
            <span class="detail-value">{{ fmt(item.amount) }} USDT</span>
          </li>
          <li>
            <span class="detail-label">{{ lang('日封顶') }}：</span>
            <span class="detail-value">{{ fmt(item.daily_cap) }} USDT</span>
          </li>
        </ul>
        <button class="purchase-btn" :disabled="loading" @click.stop="openBuy(item)">{{ lang('购买') }}</button>
      </div>
      <van-empty v-if="!loading && list.length === 0" :description="lang('暂无数据')" />
      <Pagination
        v-if="pageCount > 1"
        v-model="page"
        :page-count="pageCount"
        mode="simple"
        @change="onPageChange"
      />
    </div>
  </div>
  <Web3BuyModal
    v-model="isOpen"
    :goods-id="buyId"
    :amount="amount"
    :spot="spot"
  />
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import lang from '@/i18n/index'
import request from "@/tools/request";
import { Pagination, showFailToast } from "vant";
import { useRouter } from 'vue-router'
import { displayAmount } from '@/tools/amount'
import Web3BuyModal from './components/web3BuyModal.vue'

const pageSize = 10

const router = useRouter()
let loading = $ref(false);
const person = userPerson();
const userinfo = $computed(() => person.userinfo);
let page = $ref(1)
let total = $ref(0)
let list = $ref([])
const price = $ref('')
const isOpen = $ref(false)
const amount = $ref('')
const buyId = $ref(0)

const spot = $computed(() => price || userinfo.ispayPrice || '2000')
const pageCount = $computed(() => {
  const n = Number(total) || 0
  return n > 0 ? Math.ceil(n / pageSize) : 0
})

const fmt = (v) => displayAmount(v)

const fetchPrice = () => {
  request.get('app_server/ispay_price').then((res) => {
    if (res && res.price) price = res.price
  }).catch(() => {})
}

const fetchList = async () => {
  loading = true
  await request.get('admin/web3_goods', {
    params: {
      page,
      page_size: pageSize
    }
  }).then((res) => {
    if (res && res.status === 'ok') {
      list = Array.isArray(res.list) ? res.list : []
      total = parseInt(res.count || '0', 10) || 0
    } else {
      list = []
      total = 0
      showFailToast(res?.status || lang('暂无数据'))
    }
  }).catch(() => {
    list = []
    total = 0
  }).finally(() => {
    loading = false
  })
}

const onPageChange = (value) => {
  page = value
  fetchList()
}

const openDetail = (item) => {
  if (!item?.id) {
    showFailToast(lang('商品信息错误'))
    return
  }
  router.push(`/Web3Shop/${item.id}`)
}

const openBuy = (item) => {
  if (!item?.id) {
    showFailToast(lang('商品信息错误'))
    return
  }
  buyId = Number(item.id)
  amount = String(item.amount || '')
  isOpen = true
}

const handleBack = () => {
  router.back()
}

fetchPrice()
fetchList()
</script>
<style lang='less' scoped>
  .shop-page {
    min-height: 100vh;
    background: url('../../assets/images/topbg2.png') no-repeat;
    background-size: 100% auto;
    .page-main {
      width: 100%;
      padding: 60px 15px 30px 15px;
      box-sizing: border-box;
      .page-title {
        display: flex;
        align-items: baseline;
        gap: 8px;
        padding: 16px 0 10px;
        .price-label {
          color: #a0a0a0;
          font-size: 13px;
        }
        .price-value {
          color: #cab255;
          font-size: 18px;
          font-weight: 600;
          line-height: 1;
          em {
            font-style: normal;
            color: #9a9a9a;
            font-size: 12px;
            font-weight: 400;
            margin-left: 6px;
          }
        }
      }
      .shop-list {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
        margin-top: 10px;
        .investment-card {
          width: 100%;
          background-color: rgba(34, 34, 34, 0.88);
          border-radius: 12px;
          border: 1px solid #333;
          padding: 24px 15px;
          margin-bottom: 20px;
          box-sizing: border-box;
          cursor: pointer;
          .card-head {
            display: flex;
            align-items: flex-start;
            gap: 12px;
            margin-bottom: 8px;
          }
          .card-cover {
            width: 72px;
            height: 72px;
            flex-shrink: 0;
            border-radius: 8px;
            overflow: hidden;
            background: #2d2d2d;
            img {
              width: 100%;
              height: 100%;
              object-fit: cover;
            }
          }
          .card-head-text {
            flex: 1;
            min-width: 0;
          }
          .card-main-title {
            font-size: 18px;
            font-weight: 500;
            color: #fff;
            margin-bottom: 8px;
          }
          .card-desc {
            color: #a0a0a0;
            font-size: 13px;
            line-height: 1.5;
          }
          .investment-details {
            list-style: none;
            margin-bottom: 10px;
            li {
              display: flex;
              justify-content: space-between;
              align-items: center;
              padding: 8px 0;
              border-bottom: 1px solid #2d2d2d;
              &:last-child {
                border-bottom: none;
              }
              .detail-label {
                color: #a0a0a0;
                font-size: 14px;
              }
              .detail-value {
                color: #e0e0e0;
                font-size: 14px;
                text-align: right;
              }
            }
          }
          .purchase-btn {
            width: 100%;
            padding: 12px 20px;
            background-color: #cab255;
            color: #121212;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 500;
            margin-top: 8px;
          }
        }
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
    :deep(.van-button) {
      border-radius: 12px;
    }
  }
</style>
