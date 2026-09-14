<template>
  <a-modal forceRender :maskClosable="false" v-model:open="isOpen" :footer="null" centered destroyOnClose :title="null" @ok="handleOk">
    <div class='withdraw-dialog'>
      <div class="dialog-main">
        <div class="dialog-title">{{ lang('当前余额') }}：{{ usdtBalance }}</div>
        <a-input-number style="width: 100%" v-model:value="amount" :min="5" size="large" :placeholder="lang('请输入数量')" />
        <div class="dialog-info">
        <p><QuestionCircleOutlined style="margin-right: 5px" />{{ lang('最低充值金额') }}: 5</p>
        <p></p>
        </div>
      </div>
      <a-button class="withdraw-btn" :disabled="loading" size="large" @click="handleWithdrawal" type="primary">{{lang('充值')}}</a-button>
    </div>
  </a-modal>
</template>
<script setup>
import { QuestionCircleOutlined } from '@ant-design/icons-vue';
import userPerson from "@/pinia/person";
import fetchSign from '@/pinia/fetchSign'
import { payBuySomething } from '@/tools/buySomething'
import { showToast, showLoadingToast, closeToast, showDialog } from 'vant'
import lang from '@/i18n/index'

const person = userPerson();
const userinfo = $computed(() => person.userinfo);
const isOpen = $ref(false)
const amount = $ref(null)
const type = $ref('USDT')
const loading = $ref(false)

const props = defineProps({
  getBalance: {
    type: Function,
    required: true
  },
  onChange: {
    type: Function,
    required: true
  },
  usdtBalance: {
    type: String
  }
})

const open = (t) => {
  type = t
  isOpen = true
}

const transferUsdt = async (count) => {
  try {
    await payBuySomething(count, userinfo.buy_contract)
    loading = false;
    closeToast()
    showDialog({
      title: lang('提示'),
      message: lang(`充值成功`),
      theme: 'round-button',
      confirmButtonColor: "#242738",
      confirmButtonText: lang('我知道了！'),
    }).then(async () => {
      person.getUser();
      props.onChange()
      await props.getBalance()
      isOpen = false
    })
  } catch (error) {
    console.log(error)
    loading = false
    closeToast()
  }
}

const handleWithdrawal = async () => {
  if (loading) return

  showLoadingToast({
      message: lang('充值中'), duration: 0, overlay: true, overlayStyle: {
          background: "transparent"
      }
  });
  loading = true

  if (amount < 5) {
    loading = false
    return showToast(lang('充值金额不能小于5'))
  }
  try {
    transferUsdt(amount)
  } catch (error) {
    loading = false
  }
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
