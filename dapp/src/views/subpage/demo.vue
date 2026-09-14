<template>
<div class='page'>
  <van-nav-bar
    title="算力商城"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
  />
  <div class="page-main">
  </div>
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import { useRouter } from 'vue-router'
import lang from '@/i18n/index'

const router = useRouter()
const person = userPerson();

const handleBack = () => {
  router.back()
}

</script>
<style lang='less' scoped>
  .page {
    min-height: 100vh;
    padding-top: 50px;
  }
</style>