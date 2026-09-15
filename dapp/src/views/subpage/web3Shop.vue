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
  <div class="page-main" :class="{ 'has-cart': cartCount > 0 }">
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
          <div class="card-head-main">
            <div class="card-head-text">
              <router-link
                v-if="item.id"
                class="card-main-title"
                :to="'/Web3Shop/' + item.id"
                @click.stop
              >
                <span>{{ item.name || item.desc }}</span>
                <van-icon name="arrow" class="title-arrow" />
              </router-link>
              <div class="card-desc" v-if="item.desc && item.desc !== item.name">{{ item.desc }}</div>
            </div>
            <div class="card-foot">
              <div class="card-price">
                <span class="detail-label">{{ lang('单价') }}：</span>
                <span class="detail-value">{{ fmt(item.amount) }} USDT</span>
              </div>
              <div class="cart-actions" @click.stop>
                <div class="qty-box compact" v-if="qtyOf(item.id) > 0">
                  <button type="button" class="qty-btn" @click="changeQty(item, -1)">−</button>
                  <span class="qty-num">{{ qtyOf(item.id) }}</span>
                  <button type="button" class="qty-btn" @click="changeQty(item, 1)">+</button>
                </div>
                <button v-else type="button" class="purchase-btn" :disabled="loading" @click="addToCart(item)">{{ lang('加入购物车') }}</button>
              </div>
            </div>
          </div>
        </div>
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
  <div class="cart-bar" v-if="cartCount > 0">
    <button type="button" class="cart-icon-btn" @click="cartOpen = true">
      <van-badge :content="cartCount" max="99">
        <van-icon name="shopping-cart-o" />
      </van-badge>
    </button>
    <div class="cart-meta" @click="cartOpen = true">
      <div class="cart-total">{{ lang('合计') }} {{ fmt(cartTotal) }} <em>USDT</em></div>
      <div class="cart-cap">{{ lang('日封顶') }} {{ fmt(cartCap) }} USDT</div>
    </div>
    <button type="button" class="checkout-btn" @click="openCheckout">{{ lang('去结算') }}</button>
  </div>
  <van-popup
    v-if="cartOpen"
    v-model:show="cartOpen"
    position="bottom"
    round
    destroy-on-close
    :style="{ background: '#1a1a1a' }"
  >
    <div class="cart-sheet">
      <div class="sheet-head">
        <span>{{ lang('购物车') }}</span>
        <button type="button" class="sheet-clear" @click="clearCart">{{ lang('清空') }}</button>
      </div>
      <div class="sheet-line" v-for="row in cart" :key="row.id">
        <img v-if="row.image" :src="row.image" alt="" />
        <div class="sheet-info">
          <div class="sheet-name">{{ row.name }}</div>
          <div class="sheet-amt">{{ fmt(row.amount) }} USDT</div>
        </div>
        <div class="qty-box compact">
          <button type="button" class="qty-btn" @click="changeQty(row, -1)">−</button>
          <span class="qty-num">{{ row.qty }}</span>
          <button type="button" class="qty-btn" @click="changeQty(row, 1)">+</button>
        </div>
      </div>
      <div class="sheet-foot">
        <div class="sheet-sum">
          <div>{{ lang('合计') }} {{ fmt(cartTotal) }} USDT</div>
          <div class="cart-cap">{{ lang('日封顶') }} {{ fmt(cartCap) }} USDT</div>
        </div>
        <button type="button" class="checkout-btn" @click="openCheckout">{{ lang('去结算') }}</button>
      </div>
    </div>
  </van-popup>
  <Web3BuyModal
    v-model="isOpen"
    :items="cart"
    :amount="cartTotal"
    :spot="spot"
    @success="onBuySuccess"
    @progress="onBuyProgress"
  />
</div>
</template>
<script setup>
import { onBeforeUnmount } from 'vue'
import userPerson from "@/pinia/person";
import lang from '@/i18n/index'
import request from "@/tools/request";
import { Pagination, showFailToast } from "vant";
import { useRouter } from 'vue-router'
import { displayAmount } from '@/tools/amount'
import { capForAmount } from '@/tools/dailyCap'
import Web3BuyModal from './components/web3BuyModal.vue'

const CART_KEY = 'web3_shop_cart'

const readCart = () => {
  try {
    const raw = JSON.parse(sessionStorage.getItem(CART_KEY) || '[]')
    return Array.isArray(raw) ? raw.filter((x) => x && x.id) : []
  } catch {
    return []
  }
}

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
let cart = $ref(readCart())
let cartOpen = $ref(false)

const spot = $computed(() => price || userinfo.ispayPrice || '2000')
const cartCount = $computed(() => cart.reduce((s, x) => s + (Number(x.qty) || 0), 0))
const cartTotal = $computed(() => cart.reduce((s, x) => s + Number(x.amount || 0) * (Number(x.qty) || 0), 0))
const cartCap = $computed(() => capForAmount(cartTotal))
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
  const id = Number(item?.id)
  if (!id) {
    showFailToast(lang('商品信息错误'))
    return
  }
  router.push('/Web3Shop/' + id).catch(() => {})
}

const persistCart = () => {
  try {
    sessionStorage.setItem(CART_KEY, JSON.stringify(cart))
  } catch {}
}

const qtyOf = (id) => {
  const row = cart.find((x) => Number(x.id) === Number(id))
  return row ? Number(row.qty) || 0 : 0
}

const snapshotItem = (item) => ({
  id: Number(item.id),
  name: item.name || item.desc || '',
  image: item.image || '',
  amount: String(item.amount || ''),
  qty: 1
})

const addToCart = (item) => {
  if (!item?.id) {
    showFailToast(lang('商品信息错误'))
    return
  }
  changeQty(item, 1)
}

const changeQty = (item, delta) => {
  if (!item?.id) return
  const id = Number(item.id)
  const next = cart.map((x) => ({ ...x }))
  const i = next.findIndex((x) => Number(x.id) === id)
  if (i < 0) {
    if (delta <= 0) return
    next.push({ ...snapshotItem(item), qty: delta })
  } else {
    next[i].qty = (Number(next[i].qty) || 0) + delta
    if (next[i].qty <= 0) next.splice(i, 1)
  }
  cart = next
  persistCart()
}

const clearCart = () => {
  cart = []
  persistCart()
  cartOpen = false
}

const openCheckout = () => {
  if (!cart.length) {
    showFailToast(lang('购物车为空'))
    return
  }
  cartOpen = false
  isOpen = true
}

const onBuySuccess = () => {
  clearCart()
}

const onBuyProgress = (boughtIds) => {
  if (!Array.isArray(boughtIds) || !boughtIds.length) return
  const next = cart.map((x) => ({ ...x }))
  for (const id of boughtIds) {
    const i = next.findIndex((x) => Number(x.id) === Number(id))
    if (i < 0) continue
    next[i].qty = (Number(next[i].qty) || 0) - 1
    if (next[i].qty <= 0) next.splice(i, 1)
  }
  cart = next
  persistCart()
}

const handleBack = () => {
  cartOpen = false
  router.push('/')
}

onBeforeUnmount(() => {
  cartOpen = false
})

fetchPrice()
fetchList()
</script>
<style lang='less' scoped>
  .shop-page {
    min-height: 100vh;
    position: relative;
    background: url('../../assets/images/topbg2.png') no-repeat;
    background-size: 100% auto;
    .page-main {
      width: 100%;
      padding: 60px 15px 30px 15px;
      box-sizing: border-box;
      &.has-cart {
        padding-bottom: 110px;
      }
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
            align-items: stretch;
            gap: 14px;
          }
          .card-cover {
            width: 120px;
            height: 120px;
            flex-shrink: 0;
            border-radius: 10px;
            overflow: hidden;
            background: #2d2d2d;
            img {
              display: block;
              width: 100%;
              height: 100%;
              object-fit: cover;
            }
          }
          .card-head-main {
            flex: 1;
            min-width: 0;
            display: flex;
            flex-direction: column;
            justify-content: space-between;
          }
          .card-head-text {
            min-width: 0;
          }
          .card-foot {
            margin-top: auto;
          }
          .card-main-title {
            display: flex;
            align-items: center;
            gap: 4px;
            font-size: 18px;
            font-weight: 500;
            color: #fff;
            margin-bottom: 6px;
            line-height: 1.35;
            text-decoration: none;
            cursor: pointer;
            span {
              flex: 1;
              min-width: 0;
              overflow: hidden;
              text-overflow: ellipsis;
              white-space: nowrap;
            }
            .title-arrow {
              flex-shrink: 0;
              color: #9a9a9a;
              font-size: 16px;
              pointer-events: none;
            }
          }
          .card-desc {
            color: #a0a0a0;
            font-size: 13px;
            line-height: 1.5;
          }
          .card-price {
            display: flex;
            align-items: baseline;
            flex-wrap: wrap;
            gap: 4px;
            margin-top: 6px;
            .detail-label {
              color: #a0a0a0;
              font-size: 13px;
            }
            .detail-value {
              color: #cab255;
              font-size: 16px;
              font-weight: 600;
            }
          }
          .cart-actions {
            margin-top: 10px;
            .purchase-btn {
              width: 100%;
              padding: 8px 12px;
              background-color: #cab255;
              color: #121212;
              border: none;
              border-radius: 8px;
              font-size: 14px;
              font-weight: 500;
              line-height: 1.2;
            }
            .qty-box {
              width: 100%;
              justify-content: space-between;
            }
          }
        }
      }
    }
    .qty-box {
      display: flex;
      align-items: center;
      justify-content: space-between;
      width: 100%;
      gap: 12px;
      &.compact {
        width: auto;
        flex-shrink: 0;
        justify-content: flex-end;
        gap: 8px;
      }
    }
    .qty-btn {
      width: 32px;
      height: 32px;
      border: none;
      border-radius: 8px;
      background: #cab255;
      color: #121212;
      font-size: 18px;
      line-height: 1;
      font-weight: 600;
    }
    .qty-num {
      min-width: 24px;
      text-align: center;
      color: #fff;
      font-size: 16px;
      font-weight: 600;
    }
    .cart-bar {
      position: fixed;
      left: 0;
      right: 0;
      bottom: 0;
      z-index: 20;
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 10px 15px calc(10px + env(safe-area-inset-bottom));
      background: rgba(18, 18, 18, 0.96);
      border-top: 1px solid #333;
    }
    .cart-icon-btn {
      width: 44px;
      height: 44px;
      border: none;
      border-radius: 10px;
      background: #2d2d2d;
      color: #cab255;
      display: flex;
      align-items: center;
      justify-content: center;
      .van-icon {
        font-size: 22px;
      }
    }
    .cart-meta {
      flex: 1;
      min-width: 0;
    }
    .cart-total {
      color: #cab255;
      font-size: 16px;
      font-weight: 600;
      em {
        font-style: normal;
        font-size: 12px;
        color: #9a9a9a;
        font-weight: 400;
      }
    }
    .cart-cap {
      margin-top: 2px;
      color: #e0e0e0;
      font-size: 12px;
    }
    .checkout-btn {
      min-width: 108px;
      padding: 12px 16px;
      background-color: #cab255;
      color: #121212;
      border: none;
      border-radius: 8px;
      font-size: 16px;
      font-weight: 500;
    }
    .cart-sheet {
      padding: 16px 16px calc(16px + env(safe-area-inset-bottom));
      background: #1a1a1a;
      color: #fff;
      .sheet-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 12px;
        font-size: 16px;
        font-weight: 600;
      }
      .sheet-clear {
        border: none;
        background: transparent;
        color: #a0a0a0;
        font-size: 13px;
      }
      .sheet-line {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 10px 0;
        border-bottom: 1px solid #2d2d2d;
        img {
          width: 48px;
          height: 48px;
          border-radius: 8px;
          object-fit: cover;
          background: #2d2d2d;
          flex-shrink: 0;
        }
      }
      .sheet-info {
        flex: 1;
        min-width: 0;
      }
      .sheet-name {
        color: #fff;
        font-size: 14px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
      .sheet-amt {
        margin-top: 4px;
        color: #cab255;
        font-size: 13px;
      }
      .sheet-foot {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-top: 14px;
      }
      .sheet-sum {
        flex: 1;
        min-width: 0;
        font-size: 14px;
        font-weight: 600;
        color: #cab255;
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
    :deep(.van-popup) {
      background: #1a1a1a;
    }
    :deep(.van-badge) {
      border-color: #121212;
    }
  }
</style>
