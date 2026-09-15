<template>
  <a-modal forceRender :maskClosable="false" v-model:open="open" :footer="null" centered destroyOnClose :title="null">
    <div class="withdraw-dialog">
      <div class="dialog-main">
        <div class="dialog-title">{{ lang('购买') }}：{{ fmt(totalAmount) }} USDT</div>
        <ul class="cart-lines" v-if="lines.length > 1 || (lines[0] && lines[0].qty > 1)">
          <li v-for="line in lines" :key="line.id">
            <span>{{ line.name || (lang('商品') + ' #' + line.id) }} × {{ line.qty }}</span>
            <em>{{ fmt(Number(line.amount) * line.qty) }} USDT</em>
          </li>
        </ul>
        <p class="hint">{{ lang('充值余额') }}：{{ displayAmount(userinfo.amountUsdt) }}</p>
        <p class="hint cap-hint">{{ lang('日封顶') }}：{{ fmt(dailyCap) }} USDT<span v-if="capRange">（{{ capRange }}）</span></p>
        <p class="hint days-label">{{ lang('请选择释放天数') }}</p>
        <van-radio-group v-model="days" class="days-tabs" direction="horizontal" checked-color="#cab255" icon-size="16px">
          <van-radio
            v-for="d in dayTabs"
            :key="d"
            :name="d"
          >{{ d }}{{ lang('天') }}</van-radio>
        </van-radio-group>
        <div class="preview" v-if="days">
          <p>{{ lang('释放天数') }}：{{ days }} {{ lang('天') }}</p>
          <p>购币 {{ preview.coins }} · 日释放 {{ preview.dailyCoins }}</p>
          <p>每日约 {{ preview.usdt }} U + {{ preview.ispay }} ispay（{{ lang('现价') }} {{ spot }}）</p>
        </div>
      </div>
      <a-button class="withdraw-btn" :disabled="buying || !days || !lines.length" size="large" @click="handleBuy" type="primary">
        {{ lang('确定') }}
      </a-button>
    </div>
  </a-modal>
</template>
<script setup>
import { computed } from 'vue'
import userPerson from "@/pinia/person";
import lang from '@/i18n/index'
import request from "@/tools/request";
import { showLoadingToast, closeToast, showFailToast, showSuccessToast } from "vant";
import { displayAmount } from '@/tools/amount'
import { capForAmount, rangeForAmount, loadCapTiers, currentCapTiers } from '@/tools/dailyCap'

const dayTabs = [300, 600, 750]
const defaultTiers = [
  { days: 300, price: '1200' },
  { days: 600, price: '1000' },
  { days: 750, price: '800' }
]

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  goodsId: { type: [Number, String], default: 0 },
  amount: { type: [Number, String], default: '' },
  spot: { type: [Number, String], default: '2000' },
  items: { type: Array, default: () => [] }
})
const emit = defineEmits(['update:modelValue', 'success', 'progress'])

const person = userPerson();
const userinfo = computed(() => person.userinfo);
const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})
let days = $ref(300)
let buying = $ref(false)
const tiers = defaultTiers
let capTiers = $ref(currentCapTiers())
loadCapTiers(request).then((rows) => {
  capTiers = rows
})

const lines = $computed(() => {
  if (Array.isArray(props.items) && props.items.length) {
    return props.items
      .map((x) => ({
        id: Number(x.id),
        name: x.name || x.desc || '',
        amount: String(x.amount || '0'),
        qty: Math.max(1, Number(x.qty) || 1)
      }))
      .filter((x) => x.id > 0)
  }
  const id = Number(props.goodsId)
  if (id > 0) {
    return [{ id, name: '', amount: String(props.amount || '0'), qty: 1 }]
  }
  return []
})

const buyPayload = $computed(() => {
  if (Array.isArray(props.items) && props.items.length) {
    return {
      items: lines.map((x) => ({ id: x.id, qty: x.qty })),
      days,
      release_days: days
    }
  }
  if (!lines.length) return null
  return { id: Number(lines[0].id), days, release_days: days }
})

const totalAmount = $computed(() => {
  if (lines.length) {
    return lines.reduce((s, x) => s + Number(x.amount || 0) * Number(x.qty || 0), 0)
  }
  return Number(props.amount || 0)
})

const dailyCap = $computed(() => capForAmount(totalAmount, capTiers))
const capRange = $computed(() => rangeForAmount(totalAmount, capTiers))

const preview = $computed(() => {
  const t = tiers.find((x) => Number(x.days) === Number(days))
  const buy = Number(t?.price || 0)
  const amt = Number(totalAmount || 0)
  const d = Number(days || 0)
  const sp = Number(props.spot || 0)
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

const handleBuy = async () => {
  if (buying || !days || !lines.length || !buyPayload) return
  buying = true
  showLoadingToast()
  try {
    const res = await request.post("app_server/buy", buyPayload)
    if (res.status !== 'ok') {
      closeToast()
      showFailToast(res.status || lang('购买失败'))
      return
    }
    closeToast()
    showSuccessToast(lang('购买成功'))
    person.getUser()
    open.value = false
    emit('success')
  } catch {
    closeToast()
    showFailToast(lang('购买失败'))
  } finally {
    buying = false
  }
}
</script>
<style lang='less' scoped>
.withdraw-dialog {
  display: flex;
  flex-direction: column;
  align-items: center;
  color: #e8e8e8;
  .dialog-title {
    width: 100%;
    font-size: 16px;
    font-weight: 600;
    color: #fff;
  }
  .dialog-main {
    width: 100%;
    margin: 12px 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .preview, .hint {
    font-size: 13px;
    color: #c8c8c8;
    line-height: 1.6;
  }
  .cap-hint {
    color: #cab255;
    font-weight: 600;
  }
  .cart-lines {
    list-style: none;
    margin: 0;
    padding: 0;
    li {
      display: flex;
      justify-content: space-between;
      gap: 8px;
      font-size: 13px;
      color: #c8c8c8;
      line-height: 1.6;
      em {
        font-style: normal;
        color: #cab255;
        font-weight: 600;
      }
    }
  }
  .days-label {
    margin: 8px 0 0;
    color: #e0e0e0;
    font-weight: 600;
  }
  .days-tabs {
    display: flex;
    width: 100%;
    margin: 0 0 4px;
    padding: 4px;
    background: rgba(34, 34, 34, 0.88);
    border: 1px solid #333;
    border-radius: 12px;
    :deep(.van-radio-group) {
      display: flex;
      width: 100%;
    }
    :deep(.van-radio) {
      flex: 1;
      margin: 0;
      padding: 8px 4px;
      justify-content: center;
      border-radius: 8px;
    }
    :deep(.van-radio__label) {
      margin-left: 4px;
      color: #c8c8c8;
      font-size: 13px;
      white-space: nowrap;
    }
    :deep(.van-radio__icon .van-icon) {
      background-color: transparent;
      border-color: #666;
    }
    :deep(.van-radio__icon--checked .van-icon) {
      background-color: #cab255;
      border-color: #cab255;
      color: #121212;
    }
    :deep(.van-radio__icon--checked + .van-radio__label) {
      color: #cab255;
      font-weight: 600;
    }
    :deep(.van-radio:has(.van-radio__icon--checked)) {
      background: rgba(202, 178, 85, 0.12);
    }
  }
  .withdraw-btn {
    width: 160px;
    background-color: #cab255;
    color: #121212;
    font-weight: 500;
  }
}
</style>
