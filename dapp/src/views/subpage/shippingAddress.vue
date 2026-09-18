<template>
<div class="shop-page">
  <van-nav-bar
    :title="lang('收货地址')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
  />
  <div class="page-main">
    <div class="address-form">
      <van-field
        v-model="name"
        required
        maxlength="64"
        :label="lang('姓名')"
        :placeholder="lang('请填写姓名')"
      />
      <van-field
        v-model="contact"
        required
        maxlength="64"
        :label="lang('联系方式')"
        :placeholder="lang('请填写联系方式')"
      />
      <van-field
        v-model="address"
        required
        type="textarea"
        rows="3"
        maxlength="512"
        :label="lang('地址')"
        :placeholder="lang('请填写地址')"
      />
      <button type="button" class="save-btn" :disabled="saving" @click="save">{{ lang('保存') }}</button>
    </div>
  </div>
</div>
</template>
<script setup>
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showFailToast, showSuccessToast, showToast } from 'vant'
import lang from '@/i18n/index'
import request from '@/tools/request'
import userPerson from '@/pinia/person'

const router = useRouter()
const route = useRoute()
const person = userPerson()

let name = $ref('')
let contact = $ref('')
let address = $ref('')
let saving = $ref(false)

const fromCheckout = () => route.query.from === 'checkout'

const apply = (row) => {
  name = String(row?.name || '')
  contact = String(row?.contact || '')
  address = String(row?.address || '')
}

const load = async () => {
  try {
    const res = await request.get('app_server/shipping_address')
    if (res && res.status === 'ok') {
      apply(res)
      return
    }
  } catch {}
  apply(person.userinfo?.shippingAddress)
}

const save = async () => {
  const n = String(name || '').trim()
  const c = String(contact || '').trim()
  const a = String(address || '').trim()
  if (!n) return showToast(lang('请填写姓名'))
  if (!c) return showToast(lang('请填写联系方式'))
  if (!a) return showToast(lang('请填写地址'))
  if (saving) return
  saving = true
  try {
    const res = await request.post('app_server/shipping_address', {
      name: n,
      contact: c,
      address: a
    })
    if (res?.status !== 'ok') {
      showFailToast(res?.status || lang('操作失败'))
      return
    }
    apply(res)
    person.getUser()
    showSuccessToast(lang('保存成功'))
    if (fromCheckout()) {
      router.replace('/Web3Shop')
    }
  } catch {
    showFailToast(lang('操作失败'))
  } finally {
    saving = false
  }
}

const handleBack = () => {
  if (fromCheckout()) {
    router.replace('/Web3Shop')
    return
  }
  router.push('/')
}

onMounted(load)
</script>
<style lang="less" scoped>
.shop-page {
  min-height: 100vh;
  background: url('../../assets/images/topbg2.png') no-repeat;
  background-size: 100% auto;
  .page-main {
    width: 100%;
    padding: 72px 15px 30px;
    box-sizing: border-box;
  }
  .address-form {
    background: rgba(34, 34, 34, 0.88);
    border: 1px solid #333;
    border-radius: 12px;
    padding: 8px 8px 24px;
    --van-cell-background: transparent;
    --van-cell-text-color: #fff;
    --van-field-label-color: #cab255;
    --van-field-input-text-color: #fff;
    --van-field-placeholder-text-color: #777;
    :deep(.van-cell) {
      background: transparent;
    }
    :deep(.van-field__label) {
      color: #cab255;
    }
    :deep(.van-field__control) {
      color: #fff;
    }
  }
  .save-btn {
    display: block;
    width: 100%;
    margin-top: 20px;
    padding: 12px 16px;
    background: #cab255;
    color: #121212;
    border: none;
    border-radius: 8px;
    font-size: 16px;
    font-weight: 500;
    &:disabled {
      background: #555;
      color: #999;
    }
  }
}
</style>
