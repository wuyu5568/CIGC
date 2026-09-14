<template>
<div class='order-page'>
  <van-nav-bar
    :title="lang('订单列表')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
  />
  <div class="page-main">
    <div class="order-list" v-if="list.length > 0">
      <div class="order-item" v-for="item in list">
        <div class="order-p">
          <span>{{ item.createdAt }}</span>
          <span>{{ displayAmount(item.amount) }} USDT</span>
        </div>
        <div class="order-p">
          <span>{{ lang('已释放') }}ispay：{{ item.amountGet }}</span>
          <span>{{ lang('待释放') }}ispay：{{ item.amountLast }}</span>
        </div>
      </div>
    </div>
    <van-empty :description="lang('暂无数据')" v-else :image="emptyImage" />
    <Pagination
      v-if="allPage > 0"
      v-model="page"
      :page-count="allPage"
      mode="simple"
      @change="handlePage"
    />
  </div>
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import request from "@/tools/request";
import { useRouter, useRoute } from 'vue-router'
import { Pagination } from "vant"
import emptyImage from '../../assets/images/custom-empty-image.png'
import lang from '@/i18n/index'
import { displayAmount } from '@/tools/amount'

const route = useRoute()

const router = useRouter()
const person = userPerson();
const active = $ref('0')
const allPage = $ref(0)
const page = $ref(1)
const list = $ref([])

const getOrder = async () => {
  await request.get("app_server/order_list", {
    params: {
      page,
      orderType: active
    }
  }).then((res) => {
    const rows = Array.isArray(res) ? res : (res.list || res.items || [])
    list = rows.map((o) => ({
      createdAt: o.paid_at || o.createdAt || '',
      amount: o.amount,
      amountGet: o.release_days || '-',
      amountLast: o.title || '-'
    }))
    allPage = rows.length > 0 ? 1 : 0
  })
}

getOrder()

const changeTab = (value) => {
  list = []
  page = 1
  allPage = 0
  getOrder()
}

const handlePage = (value) => {
  page = value
  getOrder()
}

const handleBack = () => {
  router.back()
}

</script>
<style lang='less' scoped>
  .order-page {
    min-height: 100vh;
    .page-main {
      padding-top: 50px;
    }
    /deep/ .van-tab__panel {
      padding: 20px 0;
    }
    .order-list {
      display: flex;
      flex-direction: column;
      gap: 10px;
      padding: 0 15px;
      .order-item {
        display: flex;
        flex-direction: column;
        gap: 10px;
        background: #29313C;
        border-radius: 6px;
        padding: 15px;
        background: #555;
        .order-p {
          padding-bottom: 5px;
          display: flex;
          justify-content: space-between;
          color: #CCC;
        }
        .order-product {
          display: flex;
          align-items: flex-start;
          gap: 20px;
          border-bottom: 1px solid #222;
          padding-bottom: 10px;
          .order-product-image {
            width: 80px;
            height: 80px;
            background: #444;
            flex-shrink: 0;
            border-radius: 6px;
            overflow: hidden;
            img {
              width: 80px;
              height: 80px;
            }
          }
          .order-product-info {
            display: flex;
            flex-direction: column;
            gap: 15px;
            color: #CCC;
          }
        }
        .order-address {
          border-bottom: 1px solid #222;
          padding-bottom: 10px;
          color: #CCC;
          display: flex;
          flex-direction: column;
          gap: 10px;
        }
      }
      .order-footer {
        text-align: right;
        font-size: 14px;
        color: #CCC;
      }
    }
  }
</style>