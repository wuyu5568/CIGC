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
    <ul class="days-tabs">
      <li
        v-for="d in dayTabs"
        :key="d"
        :class="{ active: Number(filterDays) === d }"
        @click="onDaysChange(d)"
      >{{ d }}{{ lang('天') }}</li>
    </ul>
    <div class="shop-list">
      <div class="investment-card" v-for="item in list" :key="item.id">
        <div class="card-head">
          <div class="card-cover" v-if="item.image" @click="previewImage(item.image)">
            <img :src="item.image" alt="" />
          </div>
          <div class="card-head-text">
            <div class="card-main-title">{{ item.desc}}</div>
          </div>
        </div>
        <ul class="investment-details">
          <li>
            <span class="detail-label">{{ lang('金额') }}：</span>
            <span class="detail-value">{{ fmt(item.amount) }} USDT</span>
          </li>
          <li>
            <span class="detail-label">{{ lang('日封顶') }}：</span>
            <span class="detail-value">{{ fmt(item.daily_cap) }} USDT</span>
          </li>
        </ul>
        <button class="purchase-btn" :disabled="loading" @click="openBuy(item)">{{ lang('购买') }}</button>
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
  <a-modal forceRender :maskClosable="false" v-model:open="isOpen" :footer="null" centered destroyOnClose :title="null">
    <div class="withdraw-dialog">
      <div class="dialog-main">
        <div class="dialog-title">{{ lang('购买') }}：{{ amount }} USDT</div>
        <p class="hint">{{ lang('充值余额') }}：{{ displayAmount(userinfo.amountUsdt) }}</p>
        <p class="hint">{{ lang('释放天数') }}：{{ days }} {{ lang('天') }}</p>
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
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import lang from '@/i18n/index'
import request from "@/tools/request";
import { Pagination, showImagePreview, showLoadingToast, closeToast, showFailToast, showSuccessToast } from "vant";
import { useRouter } from 'vue-router'
import { displayAmount } from '@/tools/amount'

const dayTabs = [300, 600, 750]
const defaultTiers = [
  { days: 300, price: '1200' },
  { days: 600, price: '1000' },
  { days: 750, price: '800' }
]
const pageSize = 10

const router = useRouter()
let loading = $ref(false);
const person = userPerson();
const userinfo = $computed(() => person.userinfo);
let filterDays = $ref(300)
let page = $ref(1)
let total = $ref(0)
let list = $ref([])
const tiers = $ref(defaultTiers)
const price = $ref('')
const isOpen = $ref(false)
const amount = $ref('')
const days = $ref(null)

const spot = $computed(() => price || userinfo.ispayPrice || '2000')
const pageCount = $computed(() => {
  const n = Number(total) || 0
  return n > 0 ? Math.ceil(n / pageSize) : 0
})

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

const previewImage = (src) => {
  if (!src) return
  showImagePreview({
    images: [src],
    closeable: true
  })
}

const isAllowedDays = (value) => dayTabs.includes(Number(value))

const fetchList = async () => {
  const current = Number(filterDays)
  if (!isAllowedDays(current)) {
    list = []
    total = 0
    showFailToast(lang('请选择释放天数'))
    return
  }
  loading = true
  await request.get('admin/web3_goods', {
    params: {
      days: current,
      page,
      page_size: pageSize
    }
  }).then((res) => {
    if (res && res.status === 'ok') {
      const rows = Array.isArray(res.list) ? res.list : []
      list = rows.filter((item) => Number(item.days) === current)
      total = parseInt(res.count || '0', 10) || 0
    } else {
      list = []
      total = 0
      showFailToast(res?.status || lang('请选择释放天数'))
    }
  }).catch(() => {
    list = []
    total = 0
  }).finally(() => {
    loading = false
  })
}

const onDaysChange = (value) => {
  if (!isAllowedDays(value)) {
    showFailToast(lang('请选择释放天数'))
    return
  }
  if (Number(filterDays) === Number(value)) return
  filterDays = Number(value)
  page = 1
  isOpen = false
  fetchList()
}

const onPageChange = (value) => {
  page = value
  fetchList()
}

fetchList()

const openBuy = (item) => {
  if (!item) {
    showFailToast(lang('商品信息错误'))
    return
  }
  const itemDays = Number(item.days)
  if (!isAllowedDays(itemDays) || itemDays !== Number(filterDays)) {
    showFailToast(lang('请选择释放天数'))
    return
  }
  amount = String(item.amount || '')
  days = itemDays
  isOpen = true
}

const handleBuy = async () => {
  if (loading || !days) return
  if (!isAllowedDays(days) || Number(days) !== Number(filterDays)) {
    showFailToast(lang('请选择释放天数'))
    return
  }
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
      .days-tabs {
        display: flex;
        gap: 6px;
        padding: 4px;
        margin: 8px 0 4px;
        background: rgba(26, 26, 26, 0.88);
        border: 1px solid #333;
        border-radius: 12px;
        li {
          flex: 1;
          height: 36px;
          display: flex;
          align-items: center;
          justify-content: center;
          color: #9a9a9a;
          font-size: 14px;
          border-radius: 8px;
          &.active {
            background: #cab255;
            color: #121212;
            font-weight: 600;
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
            cursor: pointer;
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
