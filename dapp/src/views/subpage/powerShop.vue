<template>
  <div class="shop-page">
    <van-nav-bar
      :title="lang('新预售2')"
      left-arrow
      :border="false"
      :right-text="lang('订单列表')"
      fixed
      @click-left="handleBack"
      @click-right="router.push('powerOrder')"
    />
    <div class="page-main">
      <div class="page-title" style="height: 60px;">
      </div>
      <div class="shop-list">
        <div class="investment-card" v-for="item in list">
          <div class="card-main-title">{{item.minAmount}}~{{item.maxAmount}} USDT</div>
          <div class="card-price">{{ lang('价格') }}：{{ userinfo[item.tokenPrice]}} ISPAY</div>
          <ul class="investment-details">
            <li>
              <span class="detail-label">{{ lang('代币释放') }}：</span>
              <span class="detail-value">{{item.tokenReleaseRate}}</span>
            </li>
            <li>
              <span class="detail-label">{{ lang('直推奖励') }}：</span>
              <span class="detail-value">{{ item.directReferralReward }}</span>
            </li>
          </ul>
          <button class="purchase-btn" @click="handleShop(item)">{{ lang('购买') }}</button>
        </div>
        <!-- <div class="shop-item" v-for="item in userinfo.goods">
        <div class="shop-cover" :style="`box-sizing: border-box; background: url('/images/${item.three}') no-repeat; background-size: cover;  background-position: center;`"></div>
        <div class="shop-name">{{item.one}}</div>
        <div class="shop-price">${{ item.four }}</div>
        <button class="shop-btn" :disabled="loading" @click.stop @click="handleShop(item)">{{ lang('购买') }}</button>
      </div> -->
      </div>
    </div>
    <!-- <van-action-bar>
      <van-action-bar-button type="danger" :text="lang('订单列表')" @click="router.push('order')" />
    </van-action-bar> -->
    <van-action-sheet
      v-model:show="show"
      :title="lang('选择购买模式')"
      :actions="actions"
      @select="onSelect"
      :cancel-text="lang('取消')"
      close-on-click-action
    />
  </div>
  <ShopDialog ref="shopDialogRef" />
</template>
<script setup>
import userPerson from '@/pinia/person'
import lang from '@/i18n/index'
import request from '@/tools/request'
import ShopDialog from './components/shopDialog.vue'
import {
  showLoadingToast,
  closeToast,
  showConfirmDialog,
  showFailToast,
  showDialog,
  closeDialog,
  showSuccessToast
} from 'vant'
import { useRouter } from 'vue-router'

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

const packages = $ref([])
const list = $computed(() => packages.map((p) => {
  const amount = Number(p.amount)
  const days = amount <= 4000 ? 300 : amount <= 10000 ? 600 : 750
  return {
    ...p,
    minAmount: amount,
    maxAmount: amount,
    tokenPrice: 'priceOneNew',
    tokenReleaseRate: p.title || p.goods || '-',
    directReferralReward: '-',
    days
  }
}))
request.get('app_server/package_list').then((res) => {
  if (res && res.status === 'ok') {
    packages = res.items || []
  }
})

const router = useRouter()

const person = userPerson()
const userinfo = $computed(() => person.userinfo)
const sign = $computed(() => person.sign)
const id = $ref(null)
const show = $ref(false)
const shopDialogRef = $ref(null)

const handleShop = async (item) => {
  if (!item) {
    showFailToast(lang('商品信息错误'))
    return
  }
  shopDialogRef?.open(item, 'powerShop')
}

const handleBack = () => {
  router.back()
}
</script>
<style lang="less" scoped>
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
      margin-bottom: 40px;
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
      .investment-card {
        width: 100%;
        background-color: #222;
        border-radius: 12px;
        border: 1px solid #333;
        padding: 24px 15px;
        margin-bottom: 20px;
        box-sizing: border-box;
        .card-main-title {
          font-size: 18px;
          font-weight: 500;
          color: #fff;
          margin-bottom: 8px;
          display: flex;
          align-items: center;
          span {
            display: inline-block;
            width: 28px;
            height: 28px;
            line-height: 28px;
            text-align: center;
            background-color: #2d2d2d;
            color: #8ab4f8;
            border-radius: 50%;
            margin-right: 12px;
            font-size: 14px;
            font-weight: normal;
            border: 1px solid #3d3d3d;
          }
        }
        .investment-details {
          list-style: none;
          margin-bottom: 10px;
          li {
            display: flex;
            padding: 8px 0;
            border-bottom: 1px solid #2d2d2d;
            &:last-child {
              border-bottom: none;
            }
            .detail-label {
              width: 35%;
              color: #a0a0a0;
              font-size: 14px;
              font-weight: 500;
            }
            .detail-value {
              width: 65%;
              color: #e0e0e0;
              font-size: 14px;
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
          cursor: pointer;
          transition: all 0.2s ease;
          margin-top: 8px;
        }
      }
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
          background: rgba(255, 255, 255, 0.2);
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
