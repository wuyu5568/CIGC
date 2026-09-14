<template>
  <div class="profile-page">
    <ChildrenHeader />
    <ul class="profile">
      <li>
        <span>{{lang('语言切换')}}</span>
        <span>中文|English<Toggle v-model="checked" @change="onChangeLanguage" style="margin-left: 10px;"></Toggle></span>
      </li>
    </ul>
    <button class="switch-user-button" @click="switchUser">{{ lang("切换钱包") }}</button>
  </div>
</template>
<script setup lang="ts">
import ChildrenHeader from '../../components/header/childrenHeader.vue'
import { dwebServiceWorker } from "@plaoc/plugins"
import lang from '@/i18n/index'
import { useI18n } from 'vue-i18n'
import Toggle from '../../components/Toggle/Toggle.vue'
import { ref } from 'vue'
const { locale } = useI18n()

let toast: any = ref(null)

let checked = ref(localStorage.getItem('lan') === 'en' ? true : false)

const switchUser = () => {
  localStorage.removeItem("token");
  localStorage.removeItem("account");
  dwebServiceWorker.restart()
}

const onChangeLanguage = (value: boolean) => {
  if (value) {
    localStorage.setItem('lan', 'en')
    locale.value = 'en'
    location.href = '/'
  } else {
    localStorage.setItem('lan', 'zh')
    locale.value = 'zh'
    location.href = '/'
  }
}

</script>
<style scoped lang="less">
@import "./styles/index.less";

.profile-page {
  width: 100%;
  min-height: 100vh;
}
</style>
