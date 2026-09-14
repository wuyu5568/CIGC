<template>
  <a-modal forceRender :maskClosable="false" v-model:open="isOpen" :footer="null" centered destroyOnClose :title="null" @ok="handleOk">
    <div class='withdraw-dialog'>
      <div class="dialog-main">
        <div class="dialog-title">{{ lang('购买范围') }}：{{ content && content.minAmount }}~{{ content && content.maxAmount }} USDT</div>
        <a-input-number
          v-if="content"
          style="width: 100%"
          :precision="0"
          v-model:value="amount"
          :min="content.minAmount"
          :max="content.maxAmount"
          size="large"
          :placeholder="lang('请输入金额')"
          :formatter="value => `${value}`.replace(/\D/g, '')"
          :parser="value => value.replace(/\D/g, '')"
        />
      </div>
      <a-button class="withdraw-btn" :disabled="loading" size="large" @click="handleWithdrawal" type="primary">{{lang('确定')}}</a-button>
    </div>
  </a-modal>
</template>
<script setup>
import { QuestionCircleOutlined } from '@ant-design/icons-vue';
import request from "@/tools/request";
import userPerson from "@/pinia/person";
import fetchSign from '@/pinia/fetchSign'
import { showLoadingToast, showConfirmDialog, closeToast, showFailToast, showDialog, closeDialog, showSuccessToast } from "vant";
import lang from '@/i18n/index'

const person = userPerson();
const userinfo = $computed(() => person.userinfo);
const sign = $computed(() => person.sign);
const isOpen = $ref(false)
const amount = $ref(null)
const loading = $ref(false)
const content = $ref(null)
const type = $ref(null)


const props = defineProps({
  onChange: {
    type: Function,
    required: true
  }
})

const amountRound = (num) => {
  return Math.round(num * 100) / 100
}

const open = (item, newType = 'shop') => {
  // 类型校验，只允许 'shop' 或 'powerShop'
  if (!['shop', 'powerShop'].includes(newType)) {
    newType = 'shop'
  }
  type = newType
  content = item
  isOpen = true
}

const transferUsdt = async (count) => {

}

const handleWithdrawal = async () => {
  if (loading) return
  
  // 修复：请求前设置 loading
  loading = true
  
  await request.post('app_server/buy', {
    amount: amount,
    days: content && content.days ? content.days : 300,
    release_days: content && content.days ? content.days : 300,
    sign: sign
  }).then((res) => {
    if (res.status === 'ok') {
      showSuccessToast(lang('购买成功'))
      person.getUser()
      isOpen = false
    } else {
      showFailToast(res.status)
    }
  }).catch((error) => {
    showFailToast(lang('购买失败'))
  }).finally(() => {
    // 修复：请求完成后重置 loading
    loading = false
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
    background-color: #cab255;
    color: #121212;
    font-weight: 500;
  }
}
</style>