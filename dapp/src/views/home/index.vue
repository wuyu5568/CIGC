<template>
  <div class="home-page">
    <ChildrenHeader />
    <div class="pic-list">
    </div>
  </div>
</template>
<script setup lang="ts">
import ChildrenHeader from '../../components/header/childrenHeader.vue'
import userPerson from "@/pinia/person";
import lang from '@/i18n/index'
import copy from 'copy-to-clipboard';
import { computed } from 'vue'

const person = userPerson();
const userinfo = computed(() => person.userinfo);

</script>
<style scoped lang="less">
.home-page {
  width: 100%;
  min-height: 100vh;
}

.pic-list {
  img {
    width: 100%;
  }
}
</style>
