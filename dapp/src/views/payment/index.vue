<template>
  <div class="payment-page">
    <ChildrenHeader />
    <div class="payment">
      <div class="payment-card">
        <div class="payment-card-head">
          <div class="payment-card-title">钱包1</div>
          <button class="transfer-button" @click="router.push('/transfer')">钱包转账</button>
        </div>
        <div class="payment-user">
          <div class="payment-user-avatar"></div>
          <div class="payment-user-sn">
            <p>0xc84b24...ec0b19dc<span>复制</span></p>
            <p>f410fzbf...4njdiywi<span>复制</span></p>
          </div>
        </div>
        <div class="payment-user">
          <div class="payment-user-avatar"></div>
          <div class="payment-user-sn">
            <p>DIW<span>$1.2231 USD</span></p>
            <p>0.1999 DIW</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import ChildrenHeader from '../../components/header/childrenHeader.vue'
import { useRouter } from 'vue-router'
import { ref } from 'vue'

const router = useRouter()

</script>
<style scoped lang="less">
@import "./styles/index.less";

.payment-page {
  width: 100%;
  min-height: 100vh;
}
</style>
