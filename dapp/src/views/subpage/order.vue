<template>
<div class='order-page' v-if="isWeb3">
  <van-nav-bar
    :title="lang('商城订单')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
  />
  <div class="page-main">
    <div class="order-list" v-if="list.length > 0">
      <div class="order-item" v-for="item in list" :key="item.id">
        <div class="order-header">
          <span>{{ item.order_no || item.orderNo || ('#' + item.id) }}</span>
          <span>{{ statusText(item.status) }}</span>
        </div>
        <div class="order-product">
          <div class="order-product-info">
            <p>{{ item.title || item.four || '-' }}</p>
            <p>{{ item.goods || item.five || '-' }}</p>
            <p>{{ lang('套餐金额') }} {{ displayAmount(item.amount) }} USDT · {{ item.release_days }} {{ lang('天') }}</p>
            <div class="order-release">
              <p>{{ lang('购买日期') }}：{{ item.purchase_date || item.paid_at || item.createdAt || item.created_at || '-' }}</p>
              <p>{{ lang('结算日期') }}：{{ item.settle_date || '-' }}</p>
              <p>{{ lang('购币') }}：{{ displayAmount(item.coins) }} ISPAY</p>
              <p>{{ lang('已释放') }}：{{ displayAmount(item.released_coins || item.released_ispay) }} ISPAY</p>
              <p>{{ lang('待释放') }}：{{ displayAmount(item.pending_coins || item.pending_ispay) }} ISPAY</p>
              <p>{{ lang('今日释放 USDT') }}：{{ displayAmount(item.today_usdt) }}</p>
              <p>{{ lang('今日释放 ispay') }}：{{ displayAmount(item.today_ispay) }}</p>
            </div>
          </div>
        </div>
        <div class="order-footer">
          {{lang('订单金额')}}：{{ displayAmount(item.amount) }}
        </div>
      </div>
    </div>
    <van-empty :description="lang('暂无数据')" v-else :image="emptyImage" />
  </div>
</div>
<div class='order-page' v-else>
  <van-nav-bar
    :title="lang('商城订单')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
  />
  <div class="page-main">
    <van-tabs v-model:active="active" @change="changeTab">
      <van-tab :title="lang('全部')" name="0">
        <div class="order-list" v-if="list.length > 0">
          <div class="order-item" v-for="item in list">
            <div class="order-header">
              <span>{{ item.createdAt }}</span>
              <span>{{ orderTypeOption[item.orderType] }}</span>
            </div>
            <div class="order-product">
              <div class="order-product-image">
                <img :src="`/images/${item.six}`" />
              </div>
              <div class="order-product-info">
                <p>{{ item.four || '-' }}</p>
                <p>{{ item.five || '-' }}</p>
              </div>
            </div>
            <div class="order-address">
              <p>{{lang('收件人')}}：{{ item.name || '-' }}  {{ item.phone}}</p>
              <p>{{lang('收货地址')}}：{{ (item.country + item.province + item.city + item.area + item.detail) || '-' }}</p>
            </div>
            <div class="order-footer">
              {{lang('订单金额')}}：{{ displayAmount(item.amount) }}
            </div>
          </div>
        </div>
        <van-empty :description="lang('暂无数据')" v-else :image="emptyImage" />
      </van-tab>
      <van-tab :title="lang('待发货')" name="1">
        <div class="order-list" v-if="list.length > 0">
          <div class="order-item" v-for="item in list">
            <div class="order-header">
              <span>{{ item.createdAt }}</span>
              <span>{{ orderTypeOption[item.orderType] }}</span>
            </div>
            <div class="order-product">
              <div class="order-product-image">
                <img :src="`/images/${item.six}`" />
              </div>
              <div class="order-product-info">
                <p>{{ item.four || '-' }}</p>
                <p>{{ item.five || '-' }}</p>
              </div>
            </div>
            <div class="order-address">
              <p>{{lang('收件人')}}：{{ item.name || '-' }}  {{ item.phone}}</p>
              <p>{{lang('收货地址')}}：{{ (item.country + item.province + item.city + item.area + item.detail) || '-' }}</p>
            </div>
            <div class="order-footer">
              {{lang('订单金额')}}：{{ displayAmount(item.amount) }}
            </div>
          </div>
        </div>
        <van-empty :description="lang('暂无数据')" v-else :image="emptyImage" />
      </van-tab>
      <van-tab :title="lang('待收货')" name="2">
        <div class="order-list" v-if="list.length > 0">
          <div class="order-item" v-for="item in list">
            <div class="order-header">
              <span>{{ item.createdAt }}</span>
              <span>{{ orderTypeOption[item.orderType] }}</span>
            </div>
            <div class="order-product">
              <div class="order-product-image">
                <img :src="`/images/${item.six}`" />
              </div>
              <div class="order-product-info">
                <p>{{ item.four || '-' }}</p>
                <p>{{ item.five || '-' }}</p>
              </div>
            </div>
            <div class="order-address">
              <p>{{lang('收件人')}}：{{ item.name || '-' }}  {{ item.phone}}</p>
              <p>{{lang('收货地址')}}：{{ (item.country + item.province + item.city + item.area + item.detail) || '-' }}</p>
            </div>
            <div class="order-footer">
              {{lang('订单金额')}}：{{ displayAmount(item.amount) }}
            </div>
          </div>
        </div>
        <van-empty :description="lang('暂无数据')" v-else :image="emptyImage" />
      </van-tab>
      <van-tab :title="lang('已完成')" name="3">
        <div class="order-list" v-if="list.length > 0">
          <div class="order-item" v-for="item in list">
            <div class="order-header">
              <span>{{ item.createdAt }}</span>
              <span>{{ orderTypeOption[item.orderType] }}</span>
            </div>
            <div class="order-product">
              <div class="order-product-image">
                <img :src="`/images/${item.six}`" />
              </div>
              <div class="order-product-info">
                <p>{{ item.four || '-' }}</p>
                <p>{{ item.five || '-' }}</p>
              </div>
            </div>
            <div class="order-address">
              <p>{{lang('收件人')}}：{{ item.name || '-' }}  {{ item.phone}}</p>
              <p>{{lang('收货地址')}}：{{ (item.country + item.province + item.city + item.area + item.detail) || '-' }}</p>
            </div>
            <div class="order-footer">
              {{lang('订单金额')}}：{{ displayAmount(item.amount) }}
            </div>
          </div>
        </div>
        <van-empty :description="lang('暂无数据')" v-else :image="emptyImage" />
      </van-tab>
    </van-tabs>
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
const isWeb3 = $computed(() => String(route.params.id) === '1')
const active = $ref('0')
const allPage = $ref(0)
const page = $ref(1)
const list = $ref([])

console.log(route.params.id)

const orderTypeOption = {
  "1": lang('未发货'),
  "2": lang('已发货'),
  "3": lang('已送达')
}

const statusText = (s) => {
  if (s === 'paid') return lang('已支付')
  if (s === 'pending') return lang('待支付')
  return s || '-'
}

const getOrder = async () => {
  await request.get("app_server/order_list", {
    params: {
      page,
      orderType: active
    }
  }).then((res) => {
    const rows = Array.isArray(res) ? res : (res.list || res.items || [])
    list = rows.map((o) => ({
      ...o,
      createdAt: o.paid_at || o.createdAt || '',
      four: o.title,
      five: o.goods,
      orderType: o.status === 'paid' ? '3' : '1',
      name: '',
      phone: '',
      country: '',
      province: '',
      city: '',
      area: '',
      detail: '',
      six: ''
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
        .order-header {
          border-bottom: 1px solid #222;
          padding-bottom: 10px;
          display: flex;
          justify-content: space-between;
          color: #999;
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
          .order-release {
            display: flex;
            flex-direction: column;
            gap: 8px;
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