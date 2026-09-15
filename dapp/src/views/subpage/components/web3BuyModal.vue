<template>
  <a-modal forceRender :maskClosable="false" v-model:open="open" :footer="null" centered destroyOnClose :title="null">
    <div class="withdraw-dialog">
      <div class="dialog-main">
        <div class="dialog-title">{{ lang('购买') }}：{{ fmt(amount) }} USDT</div>
        <p class="hint">{{ lang('充值余额') }}：{{ displayAmount(userinfo.amountUsdt) }}</p>
        <p class="hint">{{ lang('日封顶') }}：{{ fmt(dailyCap) }} USDT</p>
        <ul class="days-tabs">
          <li
            v-for="d in dayTabs"
            :key="d"
            :class="{ active: Number(days) === d }"
            @click="days = d"
          >{{ d }}{{ lang('天') }}</li>
        </ul>
        <div class="preview" v-if="days">
          <p>{{ lang('释放天数') }}：{{ days }} {{ lang('天') }}</p>
          <p>购币 {{ preview.coins }} · 日释放 {{ preview.dailyCoins }}</p>
          <p>每日约 {{ preview.usdt }} U + {{ preview.ispay }} ispay（{{ lang('现价') }} {{ spot }}）</p>
        </div>
      </div>
      <a-button class="withdraw-btn" :disabled="buying || !days || !goodsId" size="large" @click="handleBuy" type="primary">
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
import { capForAmount } from '@/tools/dailyCap'

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
  spot: { type: [Number, String], default: '2000' }
})
const emit = defineEmits(['update:modelValue', 'success'])

const person = userPerson();
const userinfo = computed(() => person.userinfo);
const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})
let days = $ref(300)
let buying = $ref(false)
const tiers = defaultTiers

const dailyCap = $computed(() => capForAmount(props.amount))

const preview = $computed(() => {
  const t = tiers.find((x) => Number(x.days) === Number(days))
  const buy = Number(t?.price || 0)
  const amt = Number(props.amount || 0)
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
  if (buying || !days || !props.goodsId) return
  buying = true
  showLoadingToast()
  await request.post("app_server/buy", {
    id: Number(props.goodsId),
    days,
    release_days: days
  }).then((res) => {
    closeToast()
    if (res.status === 'ok') {
      showSuccessToast(lang('购买成功'))
      person.getUser()
      open.value = false
      emit('success')
    } else {
      showFailToast(res.status || lang('购买失败'))
    }
  }).catch(() => {
    closeToast()
    showFailToast(lang('购买失败'))
  }).finally(() => {
    buying = false
  })
}
</script>
<style lang='less' scoped>
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
  .days-tabs {
    display: flex;
    gap: 6px;
    padding: 4px;
    margin: 8px 0 4px;
    background: rgba(26, 26, 26, 0.08);
    border: 1px solid #eee;
    border-radius: 12px;
    li {
      flex: 1;
      height: 36px;
      display: flex;
      align-items: center;
      justify-content: center;
      color: #666;
      font-size: 14px;
      border-radius: 8px;
      cursor: pointer;
      &.active {
        background: #cab255;
        color: #121212;
        font-weight: 600;
      }
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
