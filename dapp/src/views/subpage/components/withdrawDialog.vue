<template>
  <a-modal forceRender :maskClosable="false" v-model:open="isOpen" :footer="null" centered destroyOnClose :title="null" @ok="handleOk">
    <div class='withdraw-dialog'>
      <div class="dialog-title">{{ isUsdt ? lang('当前USDT余额') : lang('当前ISPAY余额') }}：{{ fmt(balance) }}</div>
      <div class="dialog-main">
        <a-input-number style="width: 100%" v-model:value="amount" :max="Number(balance || 0)" size="large" :placeholder="lang('请输入数量')" />
        <div class="dialog-info">
        <p><QuestionCircleOutlined style="margin-right: 5px" />{{ lang('最小提现数量') }}: {{ fmt(minAmount) }}</p>
        <p>{{lang('手续费')}}：{{ fmt(fee) }}</p>
        </div>
      </div>
      <a-button class="withdraw-btn" :disabled="loading" size="large" @click="handleWithdrawal" type="primary">{{lang('提现')}}</a-button>
    </div>
  </a-modal>
</template>
<script setup>
import { QuestionCircleOutlined } from '@ant-design/icons-vue';
import userPerson from "@/pinia/person";
import fetchSign from '@/pinia/fetchSign'
import { showToast } from 'vant'
import request from "@/tools/request";
import lang from '@/i18n/index'
import { displayAmount } from '@/tools/amount'

const person = userPerson();
const userinfo = $computed(() => person.userinfo);
const isOpen = $ref(false)
const amount = $ref(null)
const type = $ref('USDT')
const loading = $ref(false)
const isUsdt = $computed(() => type === 'USDT')
const balance = $computed(() => isUsdt ? userinfo.usdt : (userinfo.ispay || userinfo.ispayAmount || 0))
const minAmount = $computed(() => isUsdt ? (userinfo.withdrawMin || 0) : (userinfo.withdrawMinTwo || 0))
const fee = $computed(() => {
  const rate = isUsdt ? userinfo.withdrawRate : userinfo.withdrawRateTwo
  return Number(rate || 0) * Number(amount || 0)
})
const fmt = (v) => displayAmount(v)

const props = defineProps({
  onChange: {
    type: Function,
    required: true
  }
})

const open = (t) => {
  type = t
  isOpen = true
}

const handleWithdrawal = async () => {
  if (loading) return
  loading = true
  if (Number(amount || 0) <= 0) {
    loading = false
    return showToast(lang('请输入金额'))
  }
  if (Number(amount) < Number(minAmount || 0)) {
    loading = false
    return showToast(lang('提现数量不能小于最小提现数量'))
  }
  if (Number(amount) > Number(balance || 0)) {
    loading = false
    return showToast(lang('提现数量不能大于余额'))
  }

  const sign = await fetchSign()

  await request.post("app_server/withdraw", {
    amount,
    sign: sign,
    coinType: isUsdt ? 1 : 3
  }).then((res) => {
    loading = false

    if (res.status === 'ok') {
      isOpen = false
      amount = null
      person.getUser()
      props.onChange()
      showToast({
        message: lang("提现成功"),
        position: 'center',
        duration: 2000,
      });
    } else {
      showToast({
        message: res.status,
        position: 'center',
        duration: 2000,
      });
		}
  }).catch((err) => {
    loading = false
    showToast({
      message: '提现失败',
      position: 'center',
      duration: 2000,
    });
  })
}

const handleOk = () => {
  console.log('WithdrawDialog handleOk')
}

defineExpose({
  open
})
</script>
<style lang="less" scoped>
.withdraw-dialog {
  display: flex;
  flex-direction: column;
  align-items: center;
  .dialog-title {
    width: 100%;
    font-size: 14px;
    font-weight: 500;;
  }
  .dialog-main {
    width: 100%;
    margin: 20px 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
    .dialog-info {
      display: flex;
      justify-content: space-between;
    }
    p {
      text-align: right;
      color: #999;
    }
  }
  .withdraw-btn {
    width: 160px;
  }
}
</style>