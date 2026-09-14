<template>
  <a-modal forceRender :maskClosable="false" v-model:open="isOpen" :footer="null" centered destroyOnClose :title="null" @ok="handleOk">
    <div class='withdraw-dialog'>
      <div class="dialog-title">{{ type === 'USDT' ? lang('当前USDT余额') : lang('当前ISPAY余额') }}：{{type === 'USDT' ? userinfo.usdt : type === 'NEWISPAY' ? userinfo.ispayAmount : userinfo.raw}}</div>
      <div class="dialog-main">
        <a-input-number style="width: 100%" v-model:value="amount" :max="type === 'USDT' ? userinfo.usdt : type === 'NEWISPAY' ? userinfo.ispayAmount : userinfo.raw" size="large" :placeholder="lang('请输入数量')" />
        <div class="dialog-info">
        <p><QuestionCircleOutlined style="margin-right: 5px" />{{ lang('最小提现数量') }}: {{userinfo.withdrawMinTwo}}</p>
        <p>{{lang('手续费')}}：{{ amountRound(Number(userinfo.withdrawRate) * (amount || 0)) }}</p>
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

const person = userPerson();
const userinfo = $computed(() => person.userinfo);
// const sign = $computed(() => person.sign);
const isOpen = $ref(false)
const amount = $ref(null)
const type = $ref('USDT')
const loading = $ref(false)

const props = defineProps({
  onChange: {
    type: Function,
    required: true
  }
})

const amountRound = (num) => {
  return Math.round(num * 100) / 100
}

const open = (t) => {
  console.log('WithdrawDialog open', t)
  type = t
  isOpen = true
}

const handleWithdrawal = async () => {
  console.log('WithdrawDialog handleWithdrawal', amount)
  if (loading) return
  loading = true
  if (amount < userinfo.withdrawMinTwo) {
    loading = false
    return showToast(lang('提现数量不能小于最小提现数量'))
  }
  if (type === 'NEWISPAY' ? amount > userinfo.ispayAmount : amount > userinfo.usdt) {
    loading = false
    return showToast(lang('提现数量不能大于余额'))
  }

  if (type === 'ISPAY') {
    loading = false
    return showToast(lang('暂不支持ISPAY提现'))
  }

  const sign = await fetchSign()

  await request.post("app_server/withdraw", {
    amount,
    sign: sign,
    coinType: type === 'NEWISPAY' ? 3 : undefined
  }).then((res) => {
    loading = false

    if (res.status === 'ok') {
      isOpen = false
      amount = null
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