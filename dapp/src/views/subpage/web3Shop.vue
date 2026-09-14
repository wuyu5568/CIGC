<template>
<div class='shop-page'>
  <van-nav-bar
    :title="lang('Web3商城')"
    :right-text="lang('收货地址')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
    @click-right="changeShippingAddress(true)"
  />
  <div class="page-main">
    <div class="page-title">
      <h2>{{ lang('Web3商城') }}</h2>
      <p>现价 {{ spot }} U / ispay · 须选 300 / 600 / 750 天</p>
    </div>
    <div class="shop-list">
      <div class="investment-card" v-for="item in list" :key="item.id">
        <div class="card-main-title">{{ item.title }} · {{ fmt(item.amount) }} USDT</div>
        <ul class="investment-details">
          <li>
            <span class="detail-label">{{ lang('商品') }}：</span>
            <span class="detail-value">{{ item.goods }}</span>
          </li>
          <li>
            <span class="detail-label">{{ lang('释放天数') }}：</span>
            <span class="detail-value">{{ item.release_days || item.days || 300 }} {{ lang('天') }}</span>
          </li>
          <li>
            <span class="detail-label">{{ lang('日封顶') }}：</span>
            <span class="detail-value">{{ fmt(item.dailyCap || item.daily_cap) }} U</span>
          </li>
        </ul>
        <button class="purchase-btn" :disabled="loading" @click="openBuy(item)">{{ lang('购买') }}</button>
      </div>
      <van-empty v-if="list.length === 0" :description="lang('暂无数据')" />
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
  <a-modal forceRender :maskClosable="false" v-model:open="isOpen" :footer="null" centered destroyOnClose :title="null">
    <div class="withdraw-dialog">
      <div class="dialog-main">
        <div class="dialog-title">{{ lang('购买') }}：{{ pkgTitle }} · {{ amount }} USDT</div>
        <p class="hint">{{ lang('充值余额') }}：{{ displayAmount(userinfo.amountUsdt) }}</p>
        <div class="dialog-label">{{ lang('释放天数') }}</div>
        <van-radio-group v-model="days">
          <van-radio v-for="t in tiers" :key="t.days" :name="t.days">
            {{ t.days }} {{ lang('天') }} · {{ lang('价格') }} {{ t.price }} U
          </van-radio>
        </van-radio-group>
        <div class="preview" v-if="days">
          <p>购币 {{ preview.coins }} · 日释放 {{ preview.dailyCoins }}</p>
          <p>每日约 {{ preview.usdt }} U + {{ preview.ispay }} ispay（现价 {{ spot }}）</p>
        </div>
      </div>
      <a-button class="withdraw-btn" :disabled="loading || !days" size="large" @click="handleBuy" type="primary">
        {{ lang('确定') }}
      </a-button>
    </div>
  </a-modal>
  <van-action-bar>
    <van-action-bar-button type="danger" :text="lang('商城订单')" @click="router.push('order/1')" />
  </van-action-bar>
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import lang from '@/i18n/index'
import request from "@/tools/request";
import { showLoadingToast, closeToast, showFailToast, showSuccessToast } from "vant";
import { useRouter } from 'vue-router'
import ShippingAddressDialog from "./components/shippingAddressDialog.vue";
import { displayAmount } from '@/tools/amount'

const defaultTiers = [
  { days: 300, price: '1200' },
  { days: 600, price: '1000' },
  { days: 750, price: '800' }
]

const router = useRouter()
let loading = $ref(false);
const person = userPerson();
const userinfo = $computed(() => person.userinfo);
const showShippingAddress = $ref(false)
const packages = $ref([])
const tiers = $ref(defaultTiers)
const price = $ref('')
const isOpen = $ref(false)
const amount = $ref('')
const pkgTitle = $ref('')
const days = $ref(null)

const spot = $computed(() => price || userinfo.ispayPrice || '2000')
const list = $computed(() => packages.length ? packages : (userinfo.goods || []))

const preview = $computed(() => {
  const t = tiers.find((x) => Number(x.days) === Number(days))
  const buy = Number(t?.price || 0)
  const amt = Number(amount || 0)
  const d = Number(days || 0)
  const sp = Number(spot || 0)
  if (!buy || !amt || !d || !sp) {
    return { coins: '-', dailyCoins: '-', usdt: '-', ispay: '-' }
  }
  const coins = amt / buy
  const dailyCoins = coins / d
  const dailyValue = dailyCoins * sp
  const usdt = dailyValue / 2
  const ispay = usdt / sp
  const r8 = (n) => displayAmount((Math.round(n * 1e8) / 1e8).toFixed(8))
  return { coins: r8(coins), dailyCoins: r8(dailyCoins), usdt: r8(usdt), ispay: r8(ispay) }
})

const fmt = (v) => displayAmount(v)

request.get('app_server/package_list').then((res) => {
  if (res && res.status === 'ok') {
    packages = res.items || []
    if (Array.isArray(res.release_tiers) && res.release_tiers.length) {
      tiers = res.release_tiers
    }
    price = res.ispay_price || ''
  }
})

const openBuy = (item) => {
  if (!item) {
    showFailToast(lang('商品信息错误'))
    return
  }
  amount = String(item.amount || '')
  pkgTitle = item.title || item.goods || ''
  days = Number(item.release_days || item.days || 300)
  isOpen = true
}

const handleBuy = async () => {
  if (loading || !days) return
  loading = true
  showLoadingToast()
  await request.post("app_server/buy", {
    amount,
    days,
    release_days: days
  }).then((res) => {
    closeToast()
    if (res.status === 'ok') {
      showSuccessToast(lang('购买成功'))
      person.getUser()
      isOpen = false
    } else {
      showFailToast(res.status || lang('购买失败'))
    }
  }).catch(() => {
    closeToast()
    showFailToast(lang('购买失败'))
  }).finally(() => {
    loading = false
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
    background: url('../../assets/images/topbg2.png') no-repeat;
    background-size: 100% auto;
    .page-main {
      width: 100%;
      padding: 60px 15px 80px 15px;
      box-sizing: border-box;
      .page-title {
        padding: 30px 0;
        h2 {
          font-size: 16px;
        }
        p {
          line-height: 1.6;
          color: #a0a0a0;
          font-size: 13px;
          margin-top: 8px;
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
            margin-top: 8px;
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
  .withdraw-dialog {
    display: flex;
    flex-direction: column;
    align-items: center;
    .dialog-title {
      width: 100%;
      font-size: 14px;
      font-weight: 500;
    }
    .dialog-label {
      margin: 12px 0 8px;
      font-size: 13px;
      color: #666;
    }
    .dialog-main {
      width: 100%;
      margin: 12px 0;
      display: flex;
      flex-direction: column;
      gap: 6px;
    }
    .preview, .hint {
      font-size: 12px;
      color: #666;
      line-height: 1.6;
    }
    .withdraw-btn {
      width: 160px;
      background-color: #cab255;
      color: #121212;
      font-weight: 500;
    }
  }
</style>
